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
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/opencontainers/go-digest"
	"gitlab.com/pantacor/pvr/templates"
)

const (
	SRC_FILE_PATH = "/src.json"
)

func (p *Pvr) GetApplicationManifest(appname string) (map[string]interface{}, error) {
	var result map[string]interface{}
	js, _, err := p.GetWorkingJson()
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(js, &result)
	if err != nil {
		return result, err
	}

	return result[appname+SRC_FILE_PATH].(map[string]interface{}), nil
}

func (p *Pvr) GenerateApplicationTemplateFiles(appname string, dockermanifest *schema2.DeserializedManifest, appmanifest map[string]interface{}) error {
	dockerConfig := dockermanifest.Config
	appConfig := appmanifest["config"]
	if appConfig != nil {
		config := appConfig.(map[string]interface{})
		if config["mediaType"] != nil {
			dockerConfig.MediaType = config["mediaType"].(string)
		}

		if config["size"] != nil {
			dockerConfig.Size = config["size"].(int64)
		}

		if config["digest"] != nil {
			dockerConfig.Digest = config["digest"].(digest.Digest)
		}

		if config["urls"] != nil {
			dockerConfig.URLs = config["urls"].([]string)
		}

		if config["annotations"] != nil {
			dockerConfig.Annotations = config["annotations"].(map[string]string)
		}
	}

	configValues := map[string]interface{}{}
	configValues["Source"] = appmanifest
	configValues["Docker"] = dockerConfig

	if appmanifest["template"] == nil {
		return fmt.Errorf("empty template")
	}

	appTemplate := appmanifest["template"].(string)
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
	appmanifest, err := p.GetApplicationManifest(appname)
	if err != nil {
		return err
	}

	if appmanifest["docker_name"] == nil {
		return err
	}

	trackURL := appmanifest["docker_name"].(string)
	if appmanifest["docker_tag"] != nil {
		trackURL += fmt.Sprintf(":%s", appmanifest["docker_tag"])
	}

	dockermanifest, err := p.GetDockerManifest(trackURL, username, password)
	if err != nil {
		return err
	}

	err = p.GenerateApplicationTemplateFiles(appname, dockermanifest, appmanifest)
	if err != nil {
		return err
	}

	destinationPath := filepath.Join(p.Dir, appname)
	return p.GenerateApplicationSquashFS(trackURL, username, password, dockermanifest, appmanifest, destinationPath)
}
