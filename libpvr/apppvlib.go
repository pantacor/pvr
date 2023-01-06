// Copyright 2022-2023  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//	Unless required by applicable law or agreed to in writing, software
//	distributed under the License is distributed on an "AS IS" BASIS,
//	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	See the License for the specific language governing permissions and
//	limitations under the License.
package libpvr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/urfave/cli"
)

func AddPvApp(p *Pvr, app *AppData) error {
	destAppPath, srcJson, err := p.GetFromRepo(app)
	if err != nil {
		return err
	}

	persistence, err := GetPersistence(app)
	if err != nil {
		return err
	}

	if app.ConfigFile != "" {
		if srcJson.DockerConfig == nil {
			srcJson.DockerConfig = map[string]interface{}{}
		}

		config, err := GetDockerConfigFile(p, app)
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

	app.Appmanifest = srcJson

	return err
}

func UpdatePvApp(p *Pvr, app *AppData, appManifest *Source) error {
	var err error

	if app.Source == "" {
		app.Source = app.Appmanifest.DockerSource.DockerSource
	}

	if app.Platform == "" {
		app.Platform = app.Appmanifest.DockerPlatform
	}

	if appManifest.PvrUrl == "" {
		return UpdateDockerApp(p, app, appManifest)
	}

	initialFrom := app.From
	app.From = appManifest.PvrUrl
	if _, appManifest, err = p.GetFromRepo(app); err != nil {
		return err
	}
	app.From = initialFrom

	srcContent, err := json.MarshalIndent(app.Appmanifest, " ", " ")
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(p.Dir, app.Appname, SRC_FILE)
	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	squashFSDigest, err := p.GetSquashFSDigest(app.SquashFile, app.Appname)
	if err != nil {
		return err
	}

	if app.Appmanifest.DockerDigest == squashFSDigest {
		fmt.Println("Application already up to date.")
		return nil
	}

	return nil
}

func InstallPVApp(p *Pvr, app *AppData, appManifest *Source) error {

	if appManifest.DockerName == "" {
		return errors.New("no docker_name in app manifest")
	}

	trackURL := appManifest.DockerName
	if appManifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", appManifest.DockerTag)
	}

	app.DockerURL = trackURL

	var dockerConfig map[string]interface{}

	if app.Appmanifest.DockerConfig != nil {
		dockerConfig = app.Appmanifest.DockerConfig
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
	err := p.GenerateApplicationTemplateFiles(app.Appname, dockerConfig, app.Appmanifest)
	if err != nil {
		return err
	}
	app.DestinationPath = filepath.Join(p.Dir, app.Appname)

	squashFSDigest, err := p.GetSquashFSDigest(app.SquashFile, app.Appname)
	if err != nil {
		return err
	}

	if app.Appmanifest.DockerDigest == squashFSDigest {
		fmt.Println("Application already up to date. Will skip generating new root.squashfs")
		return nil
	}

	return nil
}
