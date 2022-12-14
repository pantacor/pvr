// Copyright 2022  Pantacor Ltd.
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
	"os"
	"path/filepath"

	"github.com/urfave/cli"
)

func AddDockerApp(p *Pvr, app *AppData) error {
	err := p.FindDockerImage(app)
	if err != nil {
		return cli.NewExitError(err, 3)
	}

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

	persistence, err := GetPersistence(app)
	if err != nil {
		return err
	}

	src := Source{
		Spec:         SRC_SPEC,
		Template:     TEMPLATE_BUILTIN_LXC_DOCKER,
		TemplateArgs: app.TemplateArgs,
		Config:       map[string]interface{}{},
		Persistence:  persistence,
		Base:         app.Base,
		DockerSource: DockerSource{
			DockerSource:  app.Source,
			FormatOptions: app.FormatOptions,
		},
	}

	baseManifest, err := p.GetApplicationManifest(app.Base)
	if err == nil {
		src.DockerDigest = baseManifest.DockerDigest
	}

	updateDockerFromFrom(&src, app.From)

	// Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		//docker config
		src.DockerConfig = app.LocalImage.DockerConfig
		if app.DoOverlay {
			src.DockerOvlDigest = app.LocalImage.DockerDigest
		} else {
			src.DockerDigest = app.LocalImage.DockerDigest
		}
	} else if app.RemoteImage.Exists {
		// Remote repo.
		src.DockerPlatform = app.RemoteImage.DockerPlatform
		src.DockerConfig = app.RemoteImage.DockerConfig
		if app.DoOverlay {
			src.DockerOvlDigest = app.RemoteImage.DockerDigest
		} else {
			src.DockerDigest = app.RemoteImage.DockerDigest
		}
	}

	if app.ConfigFile != "" {
		dockerConfig := map[string]interface{}{}

		config, err := GetDockerConfigFile(p, app)
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

	app.Appmanifest = &src

	return p.InstallApplication(app)
}

func UpdateDockerApp(p *Pvr, app *AppData, appManifest *Source) (err error) {

	if app.Source == "" {
		app.Source = app.Appmanifest.DockerSource.DockerSource
	}
	if app.Platform == "" {
		app.Platform = app.Appmanifest.DockerPlatform
	}

	if app.From != "" {
		updateDockerFromFrom(app.Appmanifest, app.From)
	}

	err = p.FindDockerImage(app)
	if err != nil {
		return err
	}

	trackURL := app.Appmanifest.DockerName
	if app.Appmanifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", app.Appmanifest.DockerTag)
	}

	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		appManifest.DockerConfig = app.LocalImage.DockerConfig
		if app.DoOverlay {
			appManifest.DockerOvlDigest = app.LocalImage.DockerDigest
		} else {
			appManifest.DockerDigest = app.LocalImage.DockerDigest
		}
	} else if app.RemoteImage.Exists {
		appManifest.DockerPlatform = app.RemoteImage.DockerPlatform
		appManifest.DockerConfig = app.RemoteImage.DockerConfig
		if app.DoOverlay {
			appManifest.DockerOvlDigest = app.RemoteImage.DockerDigest
		} else {
			appManifest.DockerDigest = app.RemoteImage.DockerDigest
		}
	}

	for k, v := range appManifest.DockerConfig {
		if v == nil {
			delete(app.RemoteImage.DockerConfig, k)
		}
	}

	for k, v := range app.Appmanifest.DockerConfig {
		if v == nil {
			delete(app.RemoteImage.DockerConfig, k)
		}
	}

	app.Appmanifest.DockerSource.DockerSource = app.Source
	srcContent, err := json.MarshalIndent(app.Appmanifest, " ", " ")
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(p.Dir, app.Appname, SRC_FILE)
	err = ioutil.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}
	return nil
}

func InstallDockerApp(p *Pvr, app *AppData, appManifest *Source) error {
	var err error
	if appManifest.DockerName == "" {
		return errors.New("no docker_name in manifest")
	}

	trackURL := appManifest.DockerName
	if appManifest.DockerTag != "" {
		trackURL += fmt.Sprintf(":%s", appManifest.DockerTag)
	}

	app.DockerURL = trackURL

	var dockerConfig map[string]interface{}

	// src.json's generated by new pvr's will remember the DockerConfig
	// to support reapplying templates without having knowledge about the
	// the docker itself (e.g. on a plane of if the docker was never uploaded
	// to a registry). The "else" codepath exists for src.json's that were
	// generated previously. For those we will need to get dockerconfig from
	// local or remote registry still.....
	if app.Appmanifest.DockerConfig != nil {
		dockerConfig = app.Appmanifest.DockerConfig
	} else {
		err = p.FindDockerImage(app)

		if err != nil {
			fmt.Println("\nSeems like you have an invalid docker digest value in your " + app.Appname + "/src.json file\n")
			fmt.Println("\nPlease run \"pvr app update " + app.Appname + " --source=" + app.Source + "\" to auto fix it or update docker_digest field by editing " + app.Appname + "/src.json  to fix it manually\n")
			err = cli.NewExitError(err, 3)
			return err
		}

		fmt.Println("WARNING: The src.json for " + app.Appmanifest.Name + " has been genrated by old pvr; run pvr update " + app.Appmanifest.Name + " to get rid of this warning.")
		//	Exists flag is true only if the image got loaded which will depend on
		//  priority order provided in --source=local,remote
		if app.LocalImage.Exists {
			dockerConfig = app.LocalImage.DockerConfig
		} else if app.RemoteImage.Exists {
			dockerConfig = app.RemoteImage.DockerConfig
		} else {
			err = cli.NewExitError(errors.New("docker Name can not be resolved either from local docker or remote registries"), 4)
			return err
		}
	}

	err = p.GenerateApplicationTemplateFiles(app.Appname, dockerConfig, appManifest)
	if err != nil {
		return err
	}
	app.DestinationPath = filepath.Join(p.Dir, app.Appname)

	squashFSDigest, err := p.GetSquashFSDigest(app.SquashFile, app.Appname)
	if err != nil {
		return err
	}

	var baseManifest *Source
	if app.Appmanifest != nil && app.Appmanifest.Base != "" {
		baseManifest, _ = p.GetApplicationManifest(app.Appmanifest.Base)
	}

	if (app.SquashFile == SQUASH_FILE && appManifest.DockerDigest == squashFSDigest) ||
		(app.SquashFile == SQUASH_OVL_FILE && baseManifest == nil && appManifest.DockerOvlDigest == squashFSDigest) ||
		(app.SquashFile == SQUASH_OVL_FILE && baseManifest != nil && baseManifest.DockerDigest == appManifest.DockerDigest && appManifest.DockerOvlDigest == squashFSDigest) {
		fmt.Println(app.SquashFile + ": already up to date.")
		return nil
	}

	err = p.FindDockerImage(app)
	if err != nil {
		return err
	}

	fmt.Println("Generating " + app.SquashFile)

	return p.GenerateApplicationSquashFS(app, appManifest)
}
