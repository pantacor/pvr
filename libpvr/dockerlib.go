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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/api/types"
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

func (p *Pvr) GetDockerRegistry(image registry.Image, auth types.AuthConfig) (*registry.Registry, error) {
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

	auth, err := repoutils.GetAuthConfig(username, password, image.Domain)
	if err != nil {
		return nil, nil, err
	}

	r, err := p.GetDockerRegistry(image, auth)
	if err != nil {
		return nil, nil, err
	}

	manifestV2, err := r.ManifestV2(context.Background(), image.Path, image.Reference())
	if err != nil {
		return nil, nil, err
	}

	blobsURL := fmt.Sprintf("%s/v2/%s/blobs/%s", r.URL, image.Path, manifestV2.Config.Digest)

	resp, err := http.Get(blobsURL)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		wwwHeaders := resp.Header["Www-Authenticate"][0]

		// Expected format: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"

		baseReg := `%s="(([a-z]|[A-Z]|:|\/|-|_|\.)+)"`

		realmReg := regexp.MustCompile(fmt.Sprintf(baseReg, "realm"))
		realm := realmReg.FindStringSubmatch(wwwHeaders)[1]

		serviceReg := regexp.MustCompile(fmt.Sprintf(baseReg, "service"))
		service := serviceReg.FindStringSubmatch(wwwHeaders)[1]

		scopeReg := regexp.MustCompile(fmt.Sprintf(baseReg, "scope"))
		scope := scopeReg.FindStringSubmatch(wwwHeaders)[1]

		tokenURL := fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope)
		req, err := http.NewRequest(http.MethodGet, tokenURL, nil)
		if err != nil {
			return nil, nil, err
		}

		if username != "" && password != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			req.Header.Set("Authorization", "Basic "+auth)
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, nil, err
		}

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}

		var tokenResponse map[string]interface{}
		err = json.Unmarshal(content, &tokenResponse)
		if err != nil {
			return nil, nil, err
		}

		token := tokenResponse["token"].(string)
		req, err = http.NewRequest(http.MethodGet, blobsURL, nil)
		if err != nil {
			return nil, nil, err
		}

		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err = http.DefaultClient.Do(req)
	}

	if err != nil {
		return nil, nil, err
	}

	blobContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var blob map[string]interface{}
	err = json.Unmarshal(blobContent, &blob)

	config := blob["config"].(map[string]interface{})

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

	auth, err := repoutils.GetAuthConfig(username, password, image.Domain)
	if err != nil {
		return err
	}

	r, err := p.GetDockerRegistry(image, auth)
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
