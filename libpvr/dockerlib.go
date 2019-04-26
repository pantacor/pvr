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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
)

const (
	TAR_CMD            = "tar"
	MAKE_SQUASHFS_CMD  = "mksquashfs"
	SQUASH_FILE        = "root.squashfs"
	DOCKER_DIGEST_FILE = "root.squashfs.docker-digest"
)

var (
	ErrMakeSquashFSNotFound = errors.New("mksquashfs not found in your PATH, please install before continue")
	ErrTarNotFound          = errors.New("tar not found in your PATH, please install before continue")
	stripFilesList          = []string{
		"usr/bin/qemu-arm-static",
	}
)

type DockerManifest map[string]interface{}

func (p *Pvr) GetDockerRegistry(image registry.Image, username, password string) (*registry.Registry, error) {

	auth, err := repoutils.GetAuthConfig(username, password, image.Domain)
	if err != nil {
		return nil, err
	}

	return registry.New(context.Background(), auth, registry.Opt{
		Domain:   image.Domain,
		Insecure: false,
		Debug:    false,
		SkipPing: true,
		NonSSL:   false,
		Timeout:  1800 * time.Second,
	})
}

func (p *Pvr) GetDockerConfig(dockerURL, username, password string) (*schema2.Manifest, map[string]interface{}, error) {
	image, err := registry.ParseImage(dockerURL)
	if err != nil {
		return nil, nil, err
	}

	r, err := p.GetDockerRegistry(image, username, password)
	if err != nil {
		return nil, nil, err
	}

	manifestV2, err := r.ManifestV2(context.Background(), image.Path, image.Reference())
	if err != nil {
		return nil, nil, err
	}

	manifestV1, err := r.ManifestV1(context.Background(), image.Path, image.Reference())
	if err != nil {
		return nil, nil, err
	}

	config := make(map[string]interface{})
	for _, history := range manifestV1.History {
		var c map[string]interface{}
		json.Unmarshal([]byte(history.V1Compatibility), &c)
		for k, v := range c["container_config"].(map[string]interface{}) {
			config[k] = v
		}
	}

	return &manifestV2, config, nil
}

func (p *Pvr) GenerateApplicationSquashFS(dockerURL, username, password string, dockerManifest *schema2.Manifest, dockerConfig map[string]interface{}, appmanifest map[string]interface{}, destinationPath string) error {
	digestFile := filepath.Join(destinationPath, DOCKER_DIGEST_FILE)
	currentDigest, err := os.Open(digestFile)
	if err == nil {
		currentDigestContent, err := ioutil.ReadAll(currentDigest)
		if err == nil {
			if string(currentDigestContent) == string(dockerManifest.Config.Digest) {
				return nil
			}
		}
	}

	fmt.Println("Generating squashfs...")

	image, err := registry.ParseImage(dockerURL)
	if err != nil {
		return err
	}

	r, err := p.GetDockerRegistry(image, username, password)
	if err != nil {
		return err
	}

	tempdir, err := ioutil.TempDir(os.TempDir(), "download-layer-")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempdir)

	files := []string{}
	fmt.Println("Downloading layers...")
	for i, layer := range dockerManifest.Layers {
		layerReader, err := r.DownloadLayer(context.Background(), image.Path, layer.Digest)
		buf := bufio.NewReader(layerReader)

		filename := filepath.Join(tempdir, strconv.Itoa(i)) + ".tar.gz"
		file, err := os.Create(filename)
		if err != nil {
			return err
		}

		_, err = buf.WriteTo(file)
		files = append(files, filename)
		fmt.Printf("Layer %d downloaded\n", i)
	}

	extractPath := filepath.Join(tempdir, "rootfs")
	os.MkdirAll(extractPath, 0777)

	tarPath, err := exec.LookPath(TAR_CMD)
	if err != nil {
		return err
	}

	if tarPath == "" {
		return ErrTarNotFound
	}

	fmt.Println("Extracting layers...")
	for layerNumber, file := range files {
		args := []string{tarPath, "xzvf", file, "-C", extractPath}
		untar := exec.Command(args[0], args[1:]...)
		fmt.Printf("Extracting layer %d\n", layerNumber)
		untar.Run()
	}

	fmt.Println("Stripping qemu files...")
	for _, file := range stripFilesList {
		fileToDelete := filepath.Join(extractPath, file)
		os.Remove(fileToDelete)
		if err != nil {
			return err
		}

		fmt.Printf("Deleted %s file\n", fileToDelete)
	}

	makeSquashfsPath, err := exec.LookPath(MAKE_SQUASHFS_CMD)
	if err != nil {
		return err
	}

	if makeSquashfsPath == "" {
		return ErrMakeSquashFSNotFound
	}

	tempSquashFile := filepath.Join(destinationPath, SQUASH_FILE+".new")
	squashFile := filepath.Join(destinationPath, SQUASH_FILE)

	// make sure the squashfs file did not exists
	os.Remove(squashFile)

	args := []string{makeSquashfsPath, extractPath, tempSquashFile, "-comp", "xz", "-all-root"}

	fmt.Println("Generating squashfs file")
	makeSquashfs := exec.Command(args[0], args[1:]...)
	err = makeSquashfs.Run()
	if err != nil {
		return err
	}

	fmt.Println("Generating squashfs digest")

	err = os.Rename(tempSquashFile, squashFile)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(digestFile, []byte(dockerManifest.Config.Digest), 0644)
}

func (p *Pvr) GetSquashFSDigest(appName string) (string, error) {
	content, err := ioutil.ReadFile(filepath.Join(p.Dir, appName, DOCKER_DIGEST_FILE))
	if os.IsNotExist(err) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return string(content), nil
}
