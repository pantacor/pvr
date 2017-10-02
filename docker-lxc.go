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

func (p *Pvr) AddDockerLxc(dockerRegistryUrl string, dockerRepo string, dockerTag string) error {

	repoNamed, err := reference.WithName(dockerRepo)
	if err != nil {
		return err
	}

	fullRef, err := reference.WithTag(repoNamed, dockerTag)
	if err != nil {
		return err
	}

	if dockerRegistryUrl == "" {
		dockerRegistryUrl = ""
	}

	localRepoPath := path.Join(p.Dir, dockerRepo, dockerTag)
	err = os.MkdirAll(localRepoPath, 0755)
	if err != nil {
		return err
	}

	tmpdir, err := ioutil.TempDir("", "pvr-docker-lxc")
	if err != nil {
		return err
	}

	authConfig := &types.AuthConfig{}

	transport := registry.AuthTransport(nil, authConfig, false)
	httpClient := registry.HTTPClient(transport)

	session, err := registry.NewSession(httpClient, authConfig, nil)
	if err != nil {
		return err
	}

	log.Print("Openend registry Session")

	jsonB, _, err := session.GetRemoteImageJSON("library/ubuntu:latest", "https://index.docker.io/v1/")
	if err != nil {
		return err
	}

	log.Print("Retrieved Image JSON through session")

	var stringBuf string
	err = json.Unmarshal(jsonB, &stringBuf)

	if err != nil {
		return err
	}

	log.Print("JSON: " + stringBuf)

	repo, err := client.NewRepository(fullRef, dockerRegistryUrl, transport)
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
