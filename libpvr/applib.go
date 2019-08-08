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
	"path/filepath"
	"strings"

	"gitlab.com/pantacor/pvr/templates"
)

const (
	SRC_FILE                    = "src.json"
	SRC_SPEC                    = "service-manifest-src@1"
	TEMPLATE_BUILTIN_LXC_DOCKER = "builtin-lxc-docker"
)

var (
	ErrInvalidVolumeFormat = errors.New("invalid volume format")
	ErrEmptyAppName        = errors.New("empty app name")
	ErrEmptyFrom           = errors.New("empty from")
)

type Source struct {
	Name         string                 `json:"name,omitempty"`
	Spec         string                 `json:"#spec"`
	Template     string                 `json:"template"`
	TemplateArgs map[string]interface{} `json:"args"`
	Config       map[string]interface{} `json:"config"`
	DockerName   string                 `json:"docker_name"`
	DockerTag    string                 `json:"docker_tag"`
	DockerDigest string                 `json:"docker_digest"`
	Persistence  map[string]string      `json:"persistence"`
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
	}

	err = json.Unmarshal(js, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (p *Pvr) GenerateApplicationTemplateFiles(appname string, dockerConfig map[string]interface{}, appManifest *Source) error {
	appConfig := appManifest.Config
	for k, _ := range dockerConfig {
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

	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		dockerConfig = app.LocalImage.DockerConfig
	} else if app.RemoteImage.Exists {
		dockerConfig = app.RemoteImage.DockerConfig
		app.Appmanifest = appManifest
	}
	err = p.GenerateApplicationTemplateFiles(app.Appname, dockerConfig, appManifest)
	if err != nil {
		return err
	}
	app.DestinationPath = filepath.Join(p.Dir, app.Appname)

	return p.GenerateApplicationSquashFS(app)
}

func (p *Pvr) UpdateApplication(app AppData) error {
	appManifest, err := p.GetApplicationManifest(app.Appname)
	if err != nil {
		return err
	}

	trackURL := appManifest.DockerName
	if appManifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", appManifest.DockerTag)
	}

	dockerDigest := ""
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		dockerDigest = app.LocalImage.ImageID
	} else if app.RemoteImage.Exists {
		dockerDigest = app.RemoteImage.ImageID
	}

	squashFSDigest, err := p.GetSquashFSDigest(app.Appname)
	if err != nil {
		return err
	}

	if dockerDigest == squashFSDigest {
		return nil
	}

	appManifest.DockerDigest = dockerDigest

	srcContent, err := json.MarshalIndent(appManifest, " ", " ")
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(p.Dir, app.Appname, SRC_FILE)
	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	return p.InstallApplication(app)
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
	}
	components := strings.Split(app.From, ":")
	if len(components) < 2 {
		src.DockerTag = "latest"
	} else {
		src.DockerTag = components[len(components)-1]
	}
	src.DockerName = strings.Replace(app.From, ":"+src.DockerTag, "", 1)

	dockerConfig := map[string]interface{}{}
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		//docker config
		src.DockerDigest = app.LocalImage.ImageID
		dockerConfig = app.LocalImage.DockerConfig
	} else if app.RemoteImage.Exists {
		// Remote repo.
		src.DockerDigest = app.RemoteImage.ImageID
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
