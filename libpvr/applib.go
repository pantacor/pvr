//
// Copyright 2021  Pantacor Ltd.
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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"gitlab.com/pantacor/pvr/models"
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
	ErrEmptyPart           = errors.New("empty part on the pvr url PVR_URL#part")
	ErrNeedBeRoot          = errors.New("please run this command as root or use fakeroot utility")
)

type DockerSource struct {
	DockerName     string                 `json:"docker_name,omitempty"`
	DockerTag      string                 `json:"docker_tag,omitempty"`
	DockerDigest   string                 `json:"docker_digest,omitempty"`
	DockerSource   string                 `json:"docker_source,omitempty"`
	DockerConfig   map[string]interface{} `json:"docker_config,omitempty"`
	DockerPlatform string                 `json:"docker_platform,omitempty"`
	FormatOptions  string                 `json:"format_options,omitempty"`
}

type PvrSource struct {
	PvrUrl    string `json:"pvr,omitempty"`
	PvrDigest bool   `json:"pvr_digest,omitempty"`
}

type RootFsSource struct {
	RootFsURL    string `json:"rootfs_url,omitempty"`
	RootFsDigest string `json:"rootfs_digest,omitempty"`
}

type Source struct {
	Name         string                   `json:"name,omitempty"`
	Spec         string                   `json:"#spec"`
	Template     string                   `json:"template,omitempty"`
	TemplateArgs map[string]interface{}   `json:"args,omitempty"`
	Logs         []map[string]interface{} `json:"logs,omitempty"`
	Exports      []string                 `json:"exports,omitempty"`
	Config       map[string]interface{}   `json:"config,omitempty"`
	DockerSource
	PvrSource
	RootFsSource
	Persistence map[string]string `json:"persistence"`
}

// InstallApplication : Install Application from any type of source
func (p *Pvr) InstallApplication(app AppData) error {
	switch app.SourceType {
	case models.SourceTypeDocker:
		return InstallDockerApp(p, app)
	case models.SourceTypeRootFs:
		return InstallRootFsApp(p, app)
	case models.SourceTypePvr:
		return InstallPVApp(p, app)
	default:
		return fmt.Errorf("type %s not supported yet", models.SourceTypePvr)
	}
}

// UpdateApplication : Update any application and any type
func (p *Pvr) UpdateApplication(app AppData) error {
	switch app.SourceType {
	case models.SourceTypeDocker:
		return UpdateDockerApp(p, app)
	case models.SourceTypeRootFs:
		return UpdateRootFSApp(p, app)
	case models.SourceTypePvr:
		return UpdatePvApp(p, app)
	default:
		return fmt.Errorf("type %s not supported yet", models.SourceTypePvr)
	}
}

// AddApplication : Add application from several types of sources
func (p *Pvr) AddApplication(app AppData) error {
	switch app.SourceType {
	case models.SourceTypeDocker:
		return AddDockerApp(p, app)
	case models.SourceTypeRootFs:
		return AddRootFsApp(p, app)
	case models.SourceTypePvr:
		return AddPvApp(p, app)
	default:
		return fmt.Errorf("type %s not supported yet", models.SourceTypePvr)
	}
}

func (p *Pvr) isRunningAsRoot() bool {
	whoami := exec.Command("whoami")
	out, err := whoami.Output()
	if err != nil {
		whoami = exec.Command("id", "-u", "-n")
		out, err = whoami.Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error checking user id: "+err.Error())
			return false
		}
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

func (p *Pvr) SetSourceTypeFromManifest(app *AppData, options *models.GetSTOptions) error {
	if options == nil {
		options = &models.GetSTOptions{Force: false}
	}

	if app.SourceType != "" && !options.Force {
		return nil
	}

	appManifest, err := p.GetApplicationManifest(app.Appname)
	if err != nil {
		return err
	}

	app.SourceType = models.SourceTypeDocker

	if appManifest.PvrUrl != "" {
		app.SourceType = models.SourceTypePvr
	}

	if appManifest.RootFsURL != "" {
		app.SourceType = models.SourceTypeRootFs
	}

	return nil
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
	for k := range appConfig {
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
	if appManifest.TemplateArgs["PV_RUNLEVEL"] != nil && p.HasGroups() {
		fmt.Fprintln(os.Stderr, "WARNING: PV_RUNLEVEL used and groups.json found at the same time")
	}

	configValues["EffectiveGroup"] = appManifest.TemplateArgs["PV_GROUP"]

	if configValues["EffectiveGroup"] == nil && appManifest.TemplateArgs["PV_RUNLEVEL"] == nil {
		if p.GetDefaultGroup() != "" {
			configValues["EffectiveGroup"] = p.GetDefaultGroup()
		}
	}

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

func updateDockerFromFrom(src *Source, from string) {
	components := strings.Split(from, ":")
	if len(components) < 2 {
		src.DockerTag = "latest"
	} else {
		src.DockerTag = components[len(components)-1]
	}
	src.DockerName = strings.Replace(from, ":"+src.DockerTag, "", 1)
}

func (p *Pvr) GetFromRepo(app *AppData) (string, *Source, error) {
	if app.From == "" {
		return "", nil, ErrEmptyFrom
	}

	url, err := url.Parse(app.From)
	if err != nil {
		return "", nil, err
	}

	part := url.Fragment
	if part == "" {
		return "", nil, ErrEmptyPart
	}

	objectsCount, err := p.GetRepo(app.From, true, true)
	if err != nil {
		return "", nil, err
	}
	p.Pvrconfig.DefaultGetUrl = p.Pvrconfig.DefaultPostUrl
	err = p.SaveConfig()
	if err != nil {
		return "", nil, err
	}

	fmt.Println("\nImported " + strconv.Itoa(objectsCount) + " objects to " + p.Objdir)

	err = p.Reset(false)
	if err != nil {
		return "", nil, err
	}

	srcAppPath := filepath.Join(p.Dir, part)
	destAppPath := filepath.Join(p.Dir, app.Appname)
	if srcAppPath != destAppPath {
		if err = os.Rename(srcAppPath, destAppPath); err != nil {
			return "", nil, err
		}
	}

	srcContent, err := ioutil.ReadFile(filepath.Join(destAppPath, SRC_FILE))
	if err != nil {
		return "", nil, err
	}

	srcJson := Source{}
	if err = json.Unmarshal(srcContent, &srcJson); err != nil {
		return "", nil, err
	}

	srcJson.PvrSource = PvrSource{
		PvrUrl: app.From,
	}

	return destAppPath, &srcJson, nil
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

func MakeSquash(rootfsPath string, app *AppData) error {
	makeSquashfsPath, err := exec.LookPath(MAKE_SQUASHFS_CMD)
	if err != nil {
		return err
	}

	if makeSquashfsPath == "" {
		return ErrMakeSquashFSNotFound
	}

	tempSquashFile := filepath.Join(app.DestinationPath, SQUASH_FILE+".new")
	// Always Remove tempSquashfsFile
	Remove(tempSquashFile)

	squashFile := filepath.Join(app.DestinationPath, SQUASH_FILE)
	squashExist, err := IsFileExists(squashFile)
	if err != nil {
		return err
	}
	// make sure the squashfs file did not exists
	if squashExist {
		err := Remove(squashFile)
		if err != nil {
			return err
		}
	}

	var comp []string
	if app.Appmanifest.FormatOptions == "" {
		comp = []string{"-comp", "xz"}
	} else {
		comp = strings.Split(app.Appmanifest.FormatOptions, " ")
	}

	args := []string{makeSquashfsPath, rootfsPath, tempSquashFile}
	args = append(args, comp...)

	fmt.Println("Generating squashfs file: " + strings.Join(args, " "))
	makeSquashfs := exec.Command(args[0], args[1:]...)
	err = makeSquashfs.Run()
	if err != nil {
		return err
	}

	fmt.Println("Generating squashfs digest")

	return os.Rename(tempSquashFile, squashFile)
}

func GetPersistence(app *AppData) (map[string]string, error) {
	persistence := map[string]string{}
	for _, volume := range app.Volumes {
		split := strings.Split(volume, ":")
		if len(split) < 2 {
			return nil, ErrInvalidVolumeFormat
		}

		persistence[split[0]] = split[1]
	}

	return persistence, nil
}

func GetDockerConfigFile(p *Pvr, app *AppData) (map[string]interface{}, error) {
	dockerConfig := map[string]interface{}{}
	if app.ConfigFile != "" {
		configFile, err := ExpandPath(app.ConfigFile)
		if err != nil {
			return nil, err
		}

		config, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(config, &dockerConfig)
		if err != nil {
			return nil, err
		}
	}

	return dockerConfig, nil
}
