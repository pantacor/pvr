//
// Copyright 2017  Pantacor Ltd.
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
package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/cli/config"
	"github.com/docker/docker/registry"
)

func (p *Pvr) manifestToDir1(manifest *schema1.Manifest, targetDir string) error {

	log.Println("v1 only repositories not supported")
	return nil
}

func (p *Pvr) manifestToDir2(manifest *schema2.Manifest, targetDir string) error {
	descriptor := manifest.Config

	file, err := os.Open(path.Join(targetDir, ".pvr-docker-lxc.json"))
	defer file.Close()

	if err != nil {
		return err
	}

	buf, err := json.Marshal(descriptor)
	if err != nil {
		return err
	}

	_, err = file.Write(buf)

	for i, l := range manifest.Layers {
		for j, u := range l.URLs {
			log.Printf("Layer %d Url %d: %s\n", i, j, u)
		}
	}

	return err
}

func (p *Pvr) AddDockerLxc(dockerRef string, dockerServiceOptions registry.ServiceOptions) error {

	defaultService, err := registry.NewService(dockerServiceOptions)

	if err != nil {
		return err
	}

	named, err := reference.ParseNamed(dockerRef)

	if err != nil {
		return err
	}

	repoInfo, err := defaultService.ResolveRepository(named)

	if err != nil {
		return err
	}

	configDir := config.Dir()

	file, err := os.Open(path.Join(configDir, "config.json"))
	var buf []byte

	if err == nil {
		buf, err = ioutil.ReadAll(file)
	} else {
		err = nil
	}

	if err != nil {
		return err
	}

	var config map[string]interface{}

	if buf != nil {
		err = json.Unmarshal(buf, config)
	}

	var auths map[string]types.AuthConfig

	if config["auths"] != nil {
		auths = config["auths"].(map[string]types.AuthConfig)
	}

	authConfig := registry.ResolveAuthConfig(auths, repoInfo.Index)

	localRepoPath := path.Join(p.Dir, named.Name())
	err = os.MkdirAll(localRepoPath, 0755)
	if err != nil {
		return err
	}

	transport := registry.AuthTransport(nil, &authConfig, false)

	tmpdir, err := ioutil.TempDir("", "pvr-docker-lxc")
	if err != nil {
		return err
	}

	defaultService.LookupPullEndpoints(repoInfo.Index.Name)

	repo, err := client.NewRepository(named, dockerRegistryUrl, transport)
	if err != nil {
		return err
	}

	log.Print("Got Repository")

	manifestService, err := repo.Manifests(context.Background())
	if err != nil {
		return err
	}

	log.Print("Got ManifestsService")

	manifest, err := manifestService.Get(context.Background(), "", distribution.WithTag(dockerTag))

	if err != nil {
		return err
	}

	log.Print("Got Manifest")

	mediaType, buf, err := manifest.Payload()

	if err != nil {
		return err
	}

	if mediaType == schema1.MediaTypeManifest {
		var m schema1.Manifest
		err = json.Unmarshal(buf, &m)
		if err != nil {
			return err
		}
		p.manifestToDir1(&m, tmpdir)
	} else {
		var m schema2.Manifest
		err = json.Unmarshal(buf, &m)
		if err != nil {
			return err
		}
		p.manifestToDir2(&m, tmpdir)
	}

	return nil
}
