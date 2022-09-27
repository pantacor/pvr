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
	"path/filepath"

	"github.com/urfave/cli"
)

func AddPvApp(p *Pvr, app AppData) error {
	destAppPath, srcJson, err := p.GetFromRepo(&app)
	if err != nil {
		return err
	}

	persistence, err := GetPersistence(&app)
	if err != nil {
		return err
	}

	if app.ConfigFile != "" {
		if srcJson.DockerConfig == nil {
			srcJson.DockerConfig = map[string]interface{}{}
		}

		config, err := GetDockerConfigFile(p, &app)
		if err != nil {
			return err
		}

		for k, v := range config {
			srcJson.DockerConfig[k] = v
		}
	}

	srcJson.Persistence = persistence

	srcContent, err := json.MarshalIndent(srcJson, " ", " ")
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(destAppPath, SRC_FILE)
	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	return p.InstallApplication(app)
}

func UpdatePvApp(p *Pvr, app AppData) error {
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

	if appManifest.PvrUrl == "" {
		return UpdateDockerApp(p, app)
	}

	initialFrom := app.From
	app.From = appManifest.PvrUrl
	if _, appManifest, err = p.GetFromRepo(&app); err != nil {
		return err
	}
	app.From = initialFrom

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

	if appManifest.DockerDigest == squashFSDigest {
		fmt.Println("Application already up to date.")
		return nil
	}

	return p.InstallApplication(app)
}

func InstallPVApp(p *Pvr, app AppData) error {
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

	var dockerConfig map[string]interface{}

	if appManifest.DockerConfig != nil {
		dockerConfig = appManifest.DockerConfig
	} else {
		if app.LocalImage.Exists {
			dockerConfig = app.LocalImage.DockerConfig
		} else if app.RemoteImage.Exists {
			dockerConfig = app.RemoteImage.DockerConfig
		} else {
			return cli.NewExitError(errors.New("docker Name can not be resolved either from local docker or remote registries"), 4)
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

	return nil
}