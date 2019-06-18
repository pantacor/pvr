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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/go-resty/resty"
	"github.com/opencontainers/go-digest"
)

const (
	TAR_CMD                 = "tar"
	MAKE_SQUASHFS_CMD       = "mksquashfs"
	SQUASH_FILE             = "root.squashfs"
	DOCKER_DIGEST_FILE      = "root.squashfs.docker-digest"
	DOCKER_DEFAULT_REGISTRY = "registry-1.docker.io"
	DOCKER_DEFAULT_DOMAIN   = "docker.io"
	DOCKER_DEFAULT_AUTH     = "https://index.docker.io/v1/"
	DOCKER_DEFAULT_TAG      = "latest"
	DOCKER_DEFAULT_USER     = "library"

	DOCKER_MANIFEST_V2_ACCEPT_HEADER = "application/vnd.docker.distribution.manifest.v2+json"
	DOCKER_MANIFEST_V1_ACCEPT_HEADER = "application/vnd.docker.distribution.manifest.v1+prettyjws"
)

var (
	ErrMakeSquashFSNotFound = errors.New("mksquashfs not found in your PATH, please install before continue")
	ErrTarNotFound          = errors.New("tar executable not found in your PATH, please install before continue")
	ErrImageNotFound        = errors.New("image not found")
	ErrNoAccess             = errors.New("invalid username or password")
	ErrRepositoryError      = errors.New("something bad happen")

	stripFilesList = []string{
		"usr/bin/qemu-arm-static",
	}
)

type DockerManifest struct {
	DockerContentDigest string `json:"docker_content_digest"`
	SchemaVersion       int    `json:"schemaVersion"`
	MediaType           string `json:"mediaType"`
	Config              struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	} `json:"config"`
	Layers  []distribution.Descriptor `json:"layers"`
	History []struct {
		V1Compatibility string
	} `json:"history"`
}

type DockerImage struct {
	Domain string
	Path   string
	Tag    string
	Digest digest.Digest
}

type DockerLocalConfig struct {
	Auths map[string]map[string]string
}

func (p *Pvr) ParseDockerImage(dockerURL string) (DockerImage, error) {
	named, err := reference.ParseNormalizedNamed(dockerURL)
	if err != nil {
		return DockerImage{}, fmt.Errorf("parsing image %q failed: %v", dockerURL, err)
	}

	named = reference.TagNameOnly(named)

	domain := reference.Domain(named)
	if domain == DOCKER_DEFAULT_DOMAIN {
		domain = DOCKER_DEFAULT_REGISTRY
	}

	i := DockerImage{
		Domain: domain,
		Path:   reference.Path(named),
	}

	if tagged, ok := named.(reference.Tagged); ok {
		i.Tag = tagged.Tag()
	}

	if i.Tag == "" {
		i.Tag = DOCKER_DEFAULT_TAG
	}

	if canonical, ok := named.(reference.Canonical); ok {
		i.Digest = canonical.Digest()
	}

	return i, nil
}

func extractAuthURL(wwwHeaders string) string {
	baseReg := `%s="(([a-z]|[A-Z]|[0-9]|\:|\/|\-|\_|\.)+)"`

	realmReg := regexp.MustCompile(fmt.Sprintf(baseReg, "realm"))
	realm := realmReg.FindStringSubmatch(wwwHeaders)[1]

	serviceReg := regexp.MustCompile(fmt.Sprintf(baseReg, "service"))
	service := serviceReg.FindStringSubmatch(wwwHeaders)[1]

	scopeReg := regexp.MustCompile(fmt.Sprintf(baseReg, "scope"))
	scope := scopeReg.FindStringSubmatch(wwwHeaders)[1]

	return fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope)
}

func registryRequest(url, username, password string) (*resty.Response, error) {
	resty.SetRedirectPolicy(resty.FlexibleRedirectPolicy(15))
	resp, err := resty.R().
		SetHeader("Accept", DOCKER_MANIFEST_V2_ACCEPT_HEADER).
		Get(url)
	if err != nil {
		return nil, err
	}

	var usedV1Accept bool
	if resp.StatusCode() == http.StatusNotFound {
		resp, err = resty.R().
			SetHeader("Accept", DOCKER_MANIFEST_V1_ACCEPT_HEADER).
			Get(url)
		if err != nil {
			return nil, err
		}

		usedV1Accept = true
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		req := resty.R()

		if username != "" && password != "" {
			req.SetBasicAuth(username, password)
		}

		var tokenResponse map[string]interface{}
		req.SetResult(&tokenResponse)

		authURL := extractAuthURL(resp.Header().Get("Www-Authenticate"))
		resp, err = req.Get(authURL)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode() != http.StatusOK {
			log.Fatal("stop")
		}

		token := tokenResponse["token"].(string)

		req = resty.R()
		if usedV1Accept {
			req.Header.Set("Accept", DOCKER_MANIFEST_V1_ACCEPT_HEADER)
		} else {
			req.Header.Set("Accept", DOCKER_MANIFEST_V2_ACCEPT_HEADER)
		}
		req.SetAuthToken(token)

		return req.Get(url)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return nil, ErrImageNotFound
	}

	if resp.StatusCode() == http.StatusUnauthorized {
		return nil, ErrNoAccess
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, ErrRepositoryError
	}

	return resp, nil
}

func (p *Pvr) GetDockerManifest(image DockerImage, username, password string) (*DockerManifest, error) {
	manifestURL := fmt.Sprintf("https://%s/v2/%s/manifests/%s", image.Domain, image.Path, image.Tag)
	resp, err := registryRequest(manifestURL, username, password)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, errors.New(string(resp.Body()))
	}

	var manifest DockerManifest
	err = json.Unmarshal(resp.Body(), &manifest)
	if err != nil {
		return nil, err
	}

	manifest.DockerContentDigest = resp.Header().Get("Docker-Content-Digest")

	return &manifest, nil
}

func (p *Pvr) GetDockerConfig(manifest *DockerManifest, image DockerImage, username, password string) (map[string]interface{}, error) {
	if manifest.SchemaVersion == 1 {
		config := make(map[string]interface{})
		for _, c := range manifest.History {
			var hist map[string]interface{}
			err := json.Unmarshal([]byte(c.V1Compatibility), &hist)
			if err != nil {
				return nil, err
			}

			for k, v := range hist {
				if config[k] == nil {
					config[k] = v
				}
			}
		}

		return config, nil
	}

	blobsURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s", image.Domain, image.Path, manifest.Config.Digest)
	resp, err := registryRequest(blobsURL, username, password)
	if err != nil {
		return nil, err
	}

	var blob map[string]interface{}
	err = json.Unmarshal(resp.Body(), &blob)

	config := blob["config"].(map[string]interface{})

	return config, nil
}

func DownloadLayer(img *DockerImage, digest digest.Digest) (io.Reader, error) {
	downloadURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s",
		img.Domain,
		img.Path,
		digest,
	)

	resp, err := registryRequest(downloadURL, "", "")
	if err != nil {
		return nil, err
	}

	return resp.RawBody(), nil
}

func (p *Pvr) GenerateApplicationSquashFS(dockerURL, username, password string, dockerManifest *DockerManifest, dockerConfig map[string]interface{}, appmanifest map[string]interface{}, destinationPath string) error {
	digestFile := filepath.Join(destinationPath, DOCKER_DIGEST_FILE)
	currentDigest, err := os.Open(digestFile)
	if err == nil {
		currentDigestContent, err := ioutil.ReadAll(currentDigest)
		if err == nil {
			if string(currentDigestContent) == string(dockerManifest.DockerContentDigest) {
				return nil
			}
		}
	}

	fmt.Println("Generating squashfs...")

	image, err := p.ParseDockerImage(dockerURL)
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
		layerReader, err := DownloadLayer(&image, layer.Digest)
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

	return ioutil.WriteFile(digestFile, []byte(dockerManifest.DockerContentDigest), 0644)
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

func (p *Pvr) CredentialsFromCache(registry string) (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	dockerCacheFile := filepath.Join(homeDir, ".docker", "config.json")

	cacheContent, err := ioutil.ReadFile(dockerCacheFile)
	if err != nil {
		return "", "", err
	}

	var localConfig DockerLocalConfig
	err = json.Unmarshal(cacheContent, &localConfig)
	if err != nil {
		return "", "", err
	}

	creds := localConfig.Auths[registry]

	if len(creds) == 0 && registry == DOCKER_DEFAULT_REGISTRY {
		creds = localConfig.Auths[DOCKER_DEFAULT_AUTH]
	}

	if len(creds) > 0 {
		auth, err := base64.StdEncoding.DecodeString(creds["auth"])
		if err != nil {
			return "", "", err
		}

		split := strings.Split(string(auth), ":")
		return split[0], split[1], nil
	}

	return "", "", nil
}
