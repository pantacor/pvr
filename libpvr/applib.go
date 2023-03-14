//
// Copyright 2017-2023  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
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
	"gitlab.com/pantacor/pvr/utils/pvjson"
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
	DockerName      string                 `json:"docker_name,omitempty"`
	DockerTag       string                 `json:"docker_tag,omitempty"`
	DockerDigest    string                 `json:"docker_digest,omitempty"`
	DockerOvlDigest string                 `json:"docker_ovl_digest,omitempty"`
	DockerSource    string                 `json:"docker_source,omitempty"`
	DockerConfig    map[string]interface{} `json:"docker_config,omitempty"`
	DockerPlatform  string                 `json:"docker_platform,omitempty"`
	FormatOptions   string                 `json:"format_options,omitempty"`
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
	Base         string                   `json:"base,omitempty"`
	Name         string                   `json:"name,omitempty"`
	Spec         string                   `json:"#spec"`
	Template     string                   `json:"template,omitempty"`
	TemplateArgs map[string]interface{}   `json:"args,omitempty"`
	DmEnabled    map[string]bool          `json:"dm_enabled,omitempty"`
	Logs         []map[string]interface{} `json:"logs,omitempty"`
	Exports      []string                 `json:"exports,omitempty"`
	Config       map[string]interface{}   `json:"config,omitempty"`
	DockerSource
	PvrSource
	RootFsSource
	Persistence map[string]string `json:"persistence,omitempty"`
}

// InstallApplication : Install Application from any type of source
func (p *Pvr) InstallApplication(app *AppData) (err error) {

	app.SquashFile = SQUASH_FILE
	appManifest, err := p.GetApplicationManifest(app.Appname)
	if err != nil {
		return err
	}

	targetApp := *app
	targetManifest := appManifest
	if appManifest.Base != "" {
		targetManifest, err = p.GetApplicationManifest(appManifest.Base)
		if err != nil {
			return err
		}
		targetApp.Appmanifest = targetManifest
		targetApp.Appname = appManifest.Base
		targetApp.Base = ""
	}

	dockerName := targetManifest.DockerName
	repoDigest := targetManifest.DockerDigest

	targetApp.From = repoDigest
	if targetApp.Source == "remote" && !strings.Contains(repoDigest, "@") {
		targetApp.From = dockerName + "@" + repoDigest
	}

	switch targetApp.SourceType {
	case models.SourceTypeDocker:
		err = InstallDockerApp(p, &targetApp, targetManifest)
	case models.SourceTypeRootFs:
		err = InstallRootFsApp(p, &targetApp, targetManifest)
	case models.SourceTypePvr:
		err = InstallPVApp(p, &targetApp, targetManifest)
	default:
		return fmt.Errorf("type %s not supported yet", models.SourceTypePvr)
	}

	if err != nil {
		return err
	}

	// if we have ovl digest we go for it ...
	if appManifest.DockerOvlDigest != "" || appManifest.Base != "" {
		appManifest, err = p.GetApplicationManifest(app.Appname)
		if err != nil {
			return err
		}
		app.SquashFile = SQUASH_OVL_FILE

		repoDigest = appManifest.DockerOvlDigest
		app.From = repoDigest
		if app.Source == "remote" && !strings.Contains(repoDigest, "@") {
			app.From = dockerName + "@" + repoDigest
		}

		switch app.SourceType {
		case models.SourceTypeDocker:
			err = InstallDockerApp(p, app, appManifest)
		case models.SourceTypeRootFs:
			err = InstallRootFsApp(p, app, appManifest)
		case models.SourceTypePvr:
			err = InstallPVApp(p, app, appManifest)
		default:
			return fmt.Errorf("type %s not supported yet", models.SourceTypePvr)
		}
	}
	if err != nil {
		return err
	}

	diff, err := p.Diff()
	if err != nil {
		return err
	}

	// skip updating dm if nothing changed really ...
	if diff != nil && len(*diff) == 2 {
		return nil
	}

	if appManifest.DmEnabled != nil {
		err = p.DmCVerityApply(app.Appname)
	}

	return err
}

// UpdateApplication : Update any application and any type
func (p *Pvr) UpdateApplication(app AppData) error {
	appManifest, err := p.GetApplicationManifest(app.Appname)
	if err != nil {
		return err
	}

	if !app.DoOverlay && (appManifest.Base == "" || appManifest.Base == app.Appname) {
		app.SquashFile = SQUASH_FILE
		app.DoOverlay = false
		fmt.Println("Update base: " + app.SquashFile)
		appManifest.DockerOvlDigest = ""
	} else {
		app.DoOverlay = true
		app.SquashFile = SQUASH_OVL_FILE
		fmt.Println("Update ovl: " + app.SquashFile)
	}
	switch app.SourceType {
	case models.SourceTypeDocker:
		err = UpdateDockerApp(p, &app, appManifest)
	case models.SourceTypeRootFs:
		err = UpdateRootFSApp(p, &app, appManifest)
	case models.SourceTypePvr:
		err = UpdatePvApp(p, &app, appManifest)
	default:
		err = fmt.Errorf("type %s not supported yet", models.SourceTypePvr)
	}

	if err != nil {
		return err
	}

	err = p.InstallApplication(&app)
	if err != nil {
		return err
	}

	apps, err := p.GetApplications()
	if err != nil {
		return err
	}

	fmt.Printf("Searching for dependencies of %s\n", app.Appname)
	for _, a := range apps {
		if appManifest.Base == app.Appname {
			fmt.Printf("Updating dependency %s\n", a.Appname)
			if err := UpdateDockerApp(p, &a, appManifest); err != nil {
				return err
			}
			fmt.Printf("%s is up to date\n", a.Appname)
		}
	}

	return err
}

// AddApplication : Add application from several types of sources
func (p *Pvr) AddApplication(app *AppData) (err error) {
	if app.SquashFile == "" {
		app.SquashFile = SQUASH_FILE
	}
	if app.DoOverlay {
		app.SquashFile = SQUASH_OVL_FILE
	}

	switch app.SourceType {
	case models.SourceTypeDocker:
		err = AddDockerApp(p, app)
	case models.SourceTypeRootFs:
		err = AddRootFsApp(p, app)
	case models.SourceTypePvr:
		err = AddPvApp(p, app)
	default:
		err = fmt.Errorf("type %s not supported yet", models.SourceTypePvr)
	}

	if err != nil {
		return err
	}

	return p.InstallApplication(app)
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
	js, err := os.ReadFile(appManifestFile)
	if err != nil {
		return nil, err
	}

	result := Source{
		TemplateArgs: map[string]interface{}{},
		Config:       map[string]interface{}{},
		Logs:         []map[string]interface{}{},
		Exports:      []string{},
		DockerSource: DockerSource{},
	}

	err = pvjson.Unmarshal(js, &result)
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

	if appManifest.TemplateArgs["PV_GROUP"] != nil {
		fmt.Fprintf(os.Stderr, "Setting new platform to group \"%s\"\n", appManifest.TemplateArgs["PV_GROUP"].(string))
	}
	if appManifest.TemplateArgs["PV_RUNLEVEL"] != nil {
		fmt.Fprintf(os.Stderr, "Setting new platform to runlevel \"%s\"\n", appManifest.TemplateArgs["PV_RUNLEVEL"].(string))
		fmt.Fprintf(os.Stderr, "WARN: using deprecated runlevel. Use --group instead for Pantavisor 015 or above\n")
	}

	if appManifest.TemplateArgs["PV_RUNLEVEL"] != nil && p.HasGroups() {
		fmt.Fprintln(os.Stderr, "WARN: PV_RUNLEVEL used and groups.json found at the same time")
	}

	configValues["EffectiveGroup"] = appManifest.TemplateArgs["PV_GROUP"]
	if configValues["EffectiveGroup"] != nil && !p.HasGroup(configValues["EffectiveGroup"].(string)) {
		fmt.Fprintln(os.Stderr, "WARN: group does not exist in groups.json")
	}

	if configValues["EffectiveGroup"] == nil && appManifest.TemplateArgs["PV_RUNLEVEL"] == nil {
		defaultGroup := p.GetDefaultGroup()
		if defaultGroup != "" {
			fmt.Fprintf(os.Stderr, "Setting new platform to default group \"%s\"\n", defaultGroup)
			configValues["EffectiveGroup"] = defaultGroup
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
		err = os.WriteFile(filepath.Join(p.Dir, appname, name), content, 0644)
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

	parts := strings.Split(url.Fragment, ",")
	if len(parts) == 0 {
		return "", nil, ErrEmptyPart
	}

	state := PvrMap{}
	objectsCount, err := p.GetRepo(app.From, false, true, &state)
	if err != nil {
		return "", nil, err
	}
	p.Pvrconfig.DefaultGetUrl = p.Pvrconfig.DefaultPostUrl
	err = p.SaveConfig()
	if err != nil {
		return "", nil, err
	}

	fmt.Println("\nImported " + strconv.Itoa(objectsCount) + " objects to " + p.Objdir)

	err = p.ResetWithState(&state)
	if err != nil {
		return "", nil, err
	}

	srcAppPath := filepath.Join(p.Dir, parts[0])
	destAppPath := filepath.Join(p.Dir, app.Appname)
	if srcAppPath != destAppPath {
		if err = os.Rename(srcAppPath, destAppPath); err != nil {
			return "", nil, err
		}
	}

	srcContent, err := os.ReadFile(filepath.Join(destAppPath, SRC_FILE))
	if err != nil {
		return "", nil, err
	}

	srcJson := Source{}
	if err = pvjson.Unmarshal(srcContent, &srcJson); err != nil {
		return "", nil, err
	}

	srcJson.PvrSource = PvrSource{
		PvrUrl: app.From,
	}

	return destAppPath, &srcJson, nil
}

// ListApplications : List Applications
func (p *Pvr) ListApplications() error {
	files, err := os.ReadDir(p.Dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		containerConfPath := filepath.Join(p.Dir, f.Name(), "src.json")
		if _, err := os.Stat(containerConfPath); err == nil {
			fmt.Println(f.Name())
		}
	}
	return nil
}

// ListApplications : List Applications
func (p *Pvr) GetApplications() ([]AppData, error) {
	files, err := ioutil.ReadDir(p.Dir)
	if err != nil {
		return nil, err
	}

	sources := []AppData{}
	for _, f := range files {
		containerConfPath := filepath.Join(p.Dir, f.Name(), "src.json")
		if _, err := os.Stat(containerConfPath); err == nil {
			source, err := p.GetApplicationManifest(f.Name())
			if err != nil {
				return nil, err
			}
			trackURL, err := p.GetTrackURL(f.Name())
			if err != nil {
				return nil, err
			}
			sources = append(sources, AppData{
				Appmanifest:  source,
				Appname:      f.Name(),
				From:         trackURL,
				TemplateArgs: map[string]interface{}{},
			})
		}
	}
	return sources, nil
}

// GetApplicationInfo : Get Application Info
func (p *Pvr) GetApplicationInfo(appname string) error {
	srcFilePath := filepath.Join(p.Dir, appname, "src.json")
	if _, err := os.Stat(srcFilePath); err != nil {
		return errors.New("App '" + appname + "' doesn't exist")
	}
	src, _ := os.ReadFile(srcFilePath)
	var fileData interface{}
	err := pvjson.Unmarshal(src, &fileData)
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
	tempSquashFile := filepath.Join(app.DestinationPath, app.SquashFile+".new")
	squashFile := filepath.Join(app.DestinationPath, app.SquashFile)
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

		config, err := os.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		err = pvjson.Unmarshal(config, &dockerConfig)
		if err != nil {
			return nil, err
		}
	}

	return dockerConfig, nil
}
