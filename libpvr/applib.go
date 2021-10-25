//
// Copyright 2019  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//
package libpvr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/urfave/cli"
	"gitlab.com/pantacor/pvr/templates"
)

const (
	SRC_FILE                    = "src.json"
	SRC_SPEC                    = "service-manifest-src@1"
	TEMPLATE_BUILTIN_LXC_DOCKER = "builtin-lxc-docker"
)

var (
	ErrInvalidVolumeFormat = errors.New("Invalid volume format")
	ErrEmptyAppName        = errors.New("Empty app name")
	ErrEmptyFrom           = errors.New("Empty from")
	ErrNeedBeRoot          = errors.New("Please run this command as root or use fakeroot utility")
)

type DockerSource struct {
	DockerName     string                 `json:"docker_name"`
	DockerTag      string                 `json:"docker_tag"`
	DockerDigest   string                 `json:"docker_digest"`
	DockerSource   string                 `json:"docker_source"`
	DockerConfig   map[string]interface{} `json:"docker_config"`
	DockerPlatform string                 `json:"docker_platform,omitempty"`
	FormatOptions  string                 `json:"format_options,omitempty"`
}

type PvrSource struct {
	PvrUrl   string `json:"pvr_url,omitempty"`
	PvrMerge bool   `json:"pvr_merge,omitempty"`
}

type Source struct {
	Name         string                   `json:"name,omitempty"`
	Spec         string                   `json:"#spec"`
	Template     string                   `json:"template"`
	TemplateArgs map[string]interface{}   `json:"args"`
	Logs         []map[string]interface{} `json:"logs,omitempty"`
	Exports      []string                 `json:"exports,omitempty"`
	Config       map[string]interface{}   `json:"config"`
	DockerSource
	PvrSource
	Persistence map[string]string `json:"persistence"`
}

func (p *Pvr) isRunningAsRoot() bool {
	whoami := exec.Command("whoami")
	out, err := whoami.Output()
	if err != nil {
		return false
	}

	return strings.Trim(string(out), "\n") == "root"
}

func (p *Pvr) CheckIfIsRunningAsRoot() error {
	if !p.isRunningAsRoot() {
		return ErrNeedBeRoot
	}

	return nil
}

func (p *Pvr) RunAsRoot() error {
	var fakerootPath string
	fakerootPath, err := exec.LookPath("fakeroot")
	if err == nil {
		args := append([]string{fakerootPath}, os.Args...)
		return syscall.Exec(fakerootPath, args, os.Environ())
	}

	return errors.New("cannot find fakeroot in PATH. Install fakeroot or run ```pvr app``` as root: " + err.Error())
}

func (p *Pvr) GetApplicationManifest(appname string) (*Source, error) {
	appManifestFile := filepath.Join(p.Dir, appname, SRC_FILE)
	js, err := ioutil.ReadFile(appManifestFile)
	if err != nil {
		return nil, err
	}

	result := Source{
		TemplateArgs: map[string]interface{}{},
		Config:       map[string]interface{}{},
		Logs:         []map[string]interface{}{},
		Exports:      []string{},
		DockerSource: DockerSource{
			DockerSource: "remote,local",
		},
	}

	err = json.Unmarshal(js, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (p *Pvr) GenerateApplicationTemplateFiles(appname string, dockerConfig map[string]interface{}, appManifest *Source) error {
	appConfig := appManifest.Config
	for k, _ := range appConfig {
		value := appConfig[k]
		if value != nil {
			dockerConfig[k] = value
		}
	}
	// add application name to proccess template
	appManifest.Name = appname

	appManifestMap, err := StructToMap(appManifest)

	if err != nil {
		return err
	}

	configValues := map[string]interface{}{}
	configValues["Source"] = appManifestMap
	configValues["Docker"] = dockerConfig

	if appManifest.Template == "" {
		return fmt.Errorf("empty template")
	}

	appTemplate := appManifest.Template
	templateHandler := templates.Handlers[appTemplate]
	if templateHandler == nil {
		return fmt.Errorf("invalid template, no handler for %s", appTemplate)
	}

	files, err := templateHandler(configValues)
	if err != nil {
		return err
	}

	for name, content := range files {
		err = ioutil.WriteFile(filepath.Join(p.Dir, appname, name), content, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetTrackURL : Get Track URL
func (p *Pvr) GetTrackURL(appname string) (string, error) {
	appManifest, err := p.GetApplicationManifest(appname)
	if err != nil {
		return "", err
	}

	if appManifest.DockerName == "" {
		return "", err
	}

	trackURL := appManifest.DockerName
	if appManifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", appManifest.DockerTag)
	}
	return trackURL, nil
}

// GetAppDockerName : Get App Docker Name
func (p *Pvr) GetAppDockerName(appname string) (string, error) {
	appManifest, err := p.GetApplicationManifest(appname)
	if err != nil {
		return "", err
	}
	return appManifest.DockerName, nil
}

// GetAppDockerDigest : Get App Docker Digest
func (p *Pvr) GetAppDockerDigest(appname string) (string, error) {
	appManifest, err := p.GetApplicationManifest(appname)
	if err != nil {
		return "", err
	}
	return appManifest.DockerDigest, nil
}

// InstallApplication : Install Application
func (p *Pvr) InstallApplication(app AppData) error {
	appManifest, err := p.GetApplicationManifest(app.Appname)
	if err != nil {
		return err
	}

	if appManifest.DockerName == "" {
		return err
	}

	trackURL := appManifest.DockerName
	if appManifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", appManifest.DockerTag)
	}

	app.DockerURL = trackURL

	dockerConfig := map[string]interface{}{}

	// src.json's generated by new pvr's will remember the DockerConfig
	// to support reapplying templates without having knowledge about the
	// the docker itself (e.g. on a plane of if the docker was never uploaded
	// to a registry). The "else" codepath exists for src.json's that were
	// generated previously. For those we will need to get dockerconfig from
	// local or remote registry still.....
	if appManifest.DockerConfig != nil {
		dockerConfig = appManifest.DockerConfig
	} else {
		err = p.FindDockerImage(&app)

		if err != nil {
			fmt.Println("\nSeems like you have an invalid docker digest value in your " + app.Appname + "/src.json file\n")
			fmt.Println("\nPlease run \"pvr app update " + app.Appname + " --source=" + app.Source + "\" to auto fix it or update docker_digest field by editing " + app.Appname + "/src.json  to fix it manually\n")
			return cli.NewExitError(err, 3)
		}

		fmt.Println("WARNING: The src.json for " + appManifest.Name + " has been genrated by old pvr; run pvr update " + appManifest.Name + " to get rid of this warning.")
		//	Exists flag is true only if the image got loaded which will depend on
		//  priority order provided in --source=local,remote
		if app.LocalImage.Exists {
			dockerConfig = app.LocalImage.DockerConfig
		} else if app.RemoteImage.Exists {
			dockerConfig = app.RemoteImage.DockerConfig
		} else {
			return cli.NewExitError(errors.New("Docker Name can not be resolved either from local docker or remote registries."), 4)
		}
	}

	app.Appmanifest = appManifest
	err = p.GenerateApplicationTemplateFiles(app.Appname, dockerConfig, app.Appmanifest)
	if err != nil {
		return err
	}
	app.DestinationPath = filepath.Join(p.Dir, app.Appname)

	squashFSDigest, err := p.GetSquashFSDigest(app.Appname)
	if err != nil {
		return err
	}
	if appManifest.DockerDigest == squashFSDigest {
		fmt.Println("Application already up to date. Will skip generating new root.squashfs")
		return nil
	}

	return p.GenerateApplicationSquashFS(app)
}

func (p *Pvr) UpdateApplication(app AppData) error {
	appManifest, err := p.GetApplicationManifest(app.Appname)
	if err != nil {
		return err
	}
	if app.Source == "" {
		app.Source = appManifest.DockerSource.DockerSource
	}
	if app.Platform == "" {
		app.Platform = appManifest.DockerPlatform
	}

	if app.From != "" {
		updateDockerFromFrom(appManifest, app.From)
	}

	err = p.FindDockerImage(&app)
	if err != nil {
		return err
	}

	trackURL := appManifest.DockerName
	if appManifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", appManifest.DockerTag)
	}

	dockerDigest := ""
	dockerPlatform := ""
	var dockerConfig map[string]interface{}
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		dockerDigest = app.LocalImage.DockerDigest
		dockerConfig = app.LocalImage.DockerConfig
	} else if app.RemoteImage.Exists {
		dockerDigest = app.RemoteImage.DockerDigest
		dockerPlatform = app.RemoteImage.DockerPlatform
		dockerConfig = app.RemoteImage.DockerConfig
	}

	appManifest.DockerPlatform = dockerPlatform
	appManifest.DockerDigest = dockerDigest
	appManifest.DockerSource.DockerSource = app.Source
	appManifest.DockerConfig = dockerConfig

	srcContent, err := json.MarshalIndent(appManifest, " ", " ")
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(p.Dir, app.Appname, SRC_FILE)
	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	squashFSDigest, err := p.GetSquashFSDigest(app.Appname)
	if err != nil {
		return err
	}
	if dockerDigest == squashFSDigest {
		fmt.Println("Application already up to date.")
		return nil
	}
	return p.InstallApplication(app)
}

func updateDockerFromFrom(src *Source, from string) {
	components := strings.Split(from, ":")
	if len(components) < 2 {
		src.DockerTag = "latest"
	} else {
		src.DockerTag = components[len(components)-1]
	}
	src.DockerName = strings.Replace(from, ":"+src.DockerTag, "", 1)
}

func (p *Pvr) AddApplication(app AppData) error {
	if app.Appname == "" {
		return ErrEmptyAppName
	}

	appPath := filepath.Join(p.Dir, app.Appname)
	if _, err := os.Stat(appPath); !os.IsNotExist(err) {
		return nil
	}

	if app.From == "" {
		return ErrEmptyFrom
	}
	persistence := map[string]string{}
	for _, volume := range app.Volumes {
		split := strings.Split(volume, ":")
		if len(split) < 2 {
			return ErrInvalidVolumeFormat
		}

		persistence[split[0]] = split[1]
	}

	src := Source{
		Spec:         SRC_SPEC,
		Template:     TEMPLATE_BUILTIN_LXC_DOCKER,
		TemplateArgs: app.TemplateArgs,
		Config:       map[string]interface{}{},
		Persistence:  persistence,
		DockerSource: DockerSource{
			DockerSource:  app.Source,
			FormatOptions: app.FormatOptions,
		},
	}

	updateDockerFromFrom(&src, app.From)

	dockerConfig := map[string]interface{}{}
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		//docker config
		src.DockerDigest = app.LocalImage.DockerDigest
		dockerConfig = app.LocalImage.DockerConfig
	} else if app.RemoteImage.Exists {
		// Remote repo.
		src.DockerDigest = app.RemoteImage.DockerDigest
		src.DockerPlatform = app.RemoteImage.DockerPlatform
		dockerConfig = app.RemoteImage.DockerConfig
	}

	if app.ConfigFile != "" {
		var config map[string]interface{}
		content, err := ioutil.ReadFile(app.ConfigFile)
		if err != nil {
			return err
		}

		err = json.Unmarshal(content, &config)
		if err != nil {
			return err
		}

		for k, v := range config {
			dockerConfig[k] = v
		}
		//	Exists flag is true only if the image got loaded which will depend on
		//  priority order provided in --source=local,remote
		if app.LocalImage.Exists {
			app.LocalImage.DockerConfig = dockerConfig
		} else if app.RemoteImage.Exists {
			app.RemoteImage.DockerConfig = dockerConfig
		}

	}

	srcContent, err := json.MarshalIndent(src, " ", " ")
	if err != nil {
		return err
	}

	err = os.Mkdir(appPath, 0777)
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(appPath, SRC_FILE)
	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	return p.InstallApplication(app)
}

// ListApplications : List Applications
func (p *Pvr) ListApplications() error {
	files, err := ioutil.ReadDir(p.Dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		containerConfPath := filepath.Join(p.Dir, f.Name(), "lxc.container.conf")
		if _, err := os.Stat(containerConfPath); err == nil {
			fmt.Println(f.Name())
		}
	}
	return nil
}

// GetApplicationInfo : Get Application Info
func (p *Pvr) GetApplicationInfo(appname string) error {
	srcFilePath := filepath.Join(p.Dir, appname, "src.json")
	if _, err := os.Stat(srcFilePath); err != nil {
		return errors.New("App '" + appname + "' doesn't exist")
	}
	src, _ := ioutil.ReadFile(srcFilePath)
	var fileData interface{}
	err := json.Unmarshal(src, &fileData)
	if err != nil {
		return err
	}
	jsonData, err := json.MarshalIndent(fileData, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

// RemoveApplication : Remove Application
func (p *Pvr) RemoveApplication(appname string) error {
	appPath := filepath.Join(p.Dir, appname)
	if _, err := os.Stat(appPath); err != nil {
		return errors.New("App '" + appname + "' doesn't exist")
	}
	err := os.RemoveAll(appPath)
	if err != nil {
		return err
	}
	return nil
}

func ReportDockerManifestError(err error, image string) error {
	return ReportError(
		err,
		"The docker image "+image+" has to exist in local docker or in a remote registry; try docker pull "+image,
		"if the image is not public please use the --user and --password parameters or use docker login command",
	)
}
