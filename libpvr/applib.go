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
	Spec         string                 `json:"#spec"`
	Template     string                 `json:"template"`
	Config       map[string]interface{} `json:"config"`
	DockerName   string                 `json:"docker_name"`
	DockerTag    string                 `json:"docker_tag"`
	DockerDigest string                 `json:"docker_digest"`
	Persistence  map[string]string      `json:"persistence"`
}

func (p *Pvr) GetApplicationManifest(appname string) (map[string]interface{}, error) {
	appManifestFile := filepath.Join(p.Dir, appname, SRC_FILE)
	js, err := ioutil.ReadFile(appManifestFile)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(js, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p *Pvr) GenerateApplicationTemplateFiles(appname string, dockerConfig map[string]interface{}, appManifest map[string]interface{}) error {
	appConfig := appManifest["config"].(map[string]interface{})
	for k, _ := range dockerConfig {
		value := appConfig[k]
		if value != nil {
			dockerConfig[k] = value
		}
	}

	// add application name to proccess template
	appManifest["name"] = appname

	configValues := map[string]interface{}{}
	configValues["Source"] = appManifest
	configValues["Docker"] = dockerConfig

	if appManifest["template"] == nil {
		return fmt.Errorf("empty template")
	}

	appTemplate := appManifest["template"].(string)
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

func (p *Pvr) InstallApplication(appname, username, password string) error {
	appManifest, err := p.GetApplicationManifest(appname)
	if err != nil {
		return err
	}

	if appManifest["docker_name"] == nil {
		return err
	}

	trackURL := appManifest["docker_name"].(string)
	if appManifest["docker_tag"] != nil {
		trackURL += fmt.Sprintf(":%s", appManifest["docker_tag"])
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
		return ReportDockerManifestError(err)
	}

	dockerConfig, err := p.GetDockerConfig(dockerManifest, image, auth)
	if err != nil {
		return ReportDockerManifestError(err)
	}

	err = p.GenerateApplicationTemplateFiles(appname, dockerConfig, appManifest)
	if err != nil {
		return err
	}

	destinationPath := filepath.Join(p.Dir, appname)
	return p.GenerateApplicationSquashFS(trackURL, auth.Username, auth.Password, dockerManifest, dockerConfig, appManifest, destinationPath)
}

func (p *Pvr) UpdateApplication(appname, username, password string) error {
	appManifest, err := p.GetApplicationManifest(appname)
	if err != nil {
		return err
	}

	trackURL := appManifest["docker_name"].(string)
	if appManifest["docker_tag"] != nil {
		trackURL += fmt.Sprintf(":%s", appManifest["docker_tag"])
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
		return ReportDockerManifestError(err)
	}

	squashFSDigest, err := p.GetSquashFSDigest(appname)
	if err != nil {
		return err
	}

	dockerDigest := string(dockerManifest.Config.Digest)
	if dockerDigest == squashFSDigest {
		return nil
	}

	srcFilePath := filepath.Join(p.Dir, appname, SRC_FILE)
	content, err := ioutil.ReadFile(srcFilePath)
	if err != nil {
		return err
	}

	var src Source
	err = json.Unmarshal(content, &src)
	if err != nil {
		return err
	}

	src.DockerDigest = dockerDigest

	srcContent, err := json.Marshal(src)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	return p.InstallApplication(appname, auth.Username, auth.Password)
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
		return ReportDockerManifestError(err)
	}

	dockerConfig, err := p.GetDockerConfig(dockerManifest, image, auth)
	if err != nil {
		return ReportDockerManifestError(err)
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

	return p.InstallApplication(appname, username, password)
}

func ReportDockerManifestError(err error) error {
	return ReportError(
		err,
		"double check that the image exists",
		"if the image is not public please use the --user and --password parameters or use docker login command",
	)
}
