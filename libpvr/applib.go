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
	"path"
	"path/filepath"
	"strings"

	"github.com/genuinetools/reg/registry"
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
	TemplateVars map[string]interface{} `json:"vars"`
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
		TemplateVars: map[string]interface{}{},
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
func (p *Pvr) InstallApplication(
	appname string,
	username string,
	password string,
	localImage LocalDockerImage,
) error {

	appManifest, err := p.GetApplicationManifest(appname)
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
	if localImage.Exists {
		trackURL, err = p.GetSourceRepo(localImage.RepoTags, username, password)
		if err != nil {
			return err
		}
	}
	image, err := registry.ParseImage(trackURL)
	if err != nil {
		return err
	}

	auth, err := p.AuthConfig(username, password, image.Domain)
	if err != nil {
		return err
	}

	dockerManifest, err := p.GetDockerManifest(image, auth)
	if err != nil {
		return ReportDockerManifestError(err, trackURL)
	}

	dockerConfig, err := p.GetDockerConfig(dockerManifest, image, auth)
	if err != nil {
		return ReportDockerManifestError(err, trackURL)
	}

	err = p.GenerateApplicationTemplateFiles(appname, dockerConfig, appManifest)
	if err != nil {
		return err
	}

	destinationPath := filepath.Join(p.Dir, appname)
	return p.GenerateApplicationSquashFS(
		trackURL,
		auth.Username,
		auth.Password,
		dockerManifest,
		dockerConfig,
		appManifest,
		destinationPath,
		localImage,
	)
}

func (p *Pvr) UpdateApplication(appname, username, password string) error {

	appManifest, err := p.GetApplicationManifest(appname)
	if err != nil {
		return err
	}

	trackURL := appManifest.DockerName
	if appManifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", appManifest.DockerTag)
	}
	localImage, err := ImageExistsInLocalDocker(trackURL)
	if err != nil {
		return err
	}
	if localImage.Exists {
		trackURL, err = p.GetSourceRepo(localImage.RepoTags, username, password)
		if err != nil {
			return err
		}
	}
	image, err := registry.ParseImage(trackURL)
	if err != nil {
		return err
	}

	auth, err := p.AuthConfig(username, password, image.Domain)
	if err != nil {
		return err
	}

	dockerManifest, err := p.GetDockerManifest(image, auth)
	if err != nil {
		return ReportDockerManifestError(err, trackURL)
	}

	squashFSDigest, err := p.GetSquashFSDigest(appname)
	if err != nil {
		return err
	}

	dockerDigest := string(dockerManifest.Config.Digest)
	if dockerDigest == squashFSDigest {
		return nil
	}

	appManifest.DockerDigest = dockerDigest

	srcContent, err := json.MarshalIndent(appManifest, " ", " ")
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(p.Dir, appname, SRC_FILE)
	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	return p.InstallApplication(appname, auth.Username, auth.Password, localImage)
}

func (p *Pvr) AddApplication(appname, username, password, from, configFile string, volumes []string) error {
	if appname == "" {
		return ErrEmptyAppName
	}

	appPath := filepath.Join(p.Dir, appname)
	if _, err := os.Stat(appPath); !os.IsNotExist(err) {
		return nil
	}

	if from == "" {
		return ErrEmptyFrom
	}
	localImage, err := ImageExistsInLocalDocker(from)
	if err != nil {
		return err
	}
	if localImage.Exists {
		from, err = p.GetSourceRepo(localImage.RepoTags, username, password)
		if err != nil {
			return err
		}
	}

	image, err := registry.ParseImage(from)
	if err != nil {
		return err
	}

	auth, err := p.AuthConfig(username, password, image.Domain)
	if err != nil {
		return err
	}

	dockerManifest, err := p.GetDockerManifest(image, auth)
	if err != nil {
		return ReportDockerManifestError(err, from)
	}
	dockerConfig, err := p.GetDockerConfig(dockerManifest, image, auth)
	if err != nil {
		return ReportDockerManifestError(err, from)
	}

	dockerDigest := string(dockerManifest.Config.Digest)

	if configFile != "" {
		var config map[string]interface{}
		content, err := ioutil.ReadFile(configFile)
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

	}

	persistence := map[string]string{}
	for _, volume := range volumes {
		split := strings.Split(volume, ":")
		if len(split) < 2 {
			return ErrInvalidVolumeFormat
		}

		persistence[split[0]] = split[1]
	}

	src := Source{
		Spec:         SRC_SPEC,
		Template:     TEMPLATE_BUILTIN_LXC_DOCKER,
		TemplateVars: map[string]interface{}{},
		Config:       map[string]interface{}{},
		DockerName:   path.Join(image.Domain, image.Path),
		DockerTag:    image.Tag,
		DockerDigest: string(dockerDigest),
		Persistence:  persistence,
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

	return p.InstallApplication(appname, username, password, localImage)
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
