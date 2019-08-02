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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
)

const (
	TAR_CMD                        = "tar"
	MAKE_SQUASHFS_CMD              = "mksquashfs"
	SQUASH_FILE                    = "root.squashfs"
	DOCKER_DIGEST_FILE             = "root.squashfs.docker-digest"
	DOCKER_DOMAIN                  = "docker.io"
	DOCKER_DOMAIN_URL              = "https://" + DOCKER_DOMAIN
	DOCKER_REGISTRY                = "https://index.docker.io/v1/"
	DOCKER_REGISTRY_SERVER_ADDRESS = "https://registry-1.docker.io"
)

var (
	ErrMakeSquashFSNotFound    = errors.New("mksquashfs not found in your PATH, please install before continue")
	ErrTarNotFound             = errors.New("tar not found in your PATH, please install before continue")
	ErrImageNotFound           = errors.New("image not found or you do not have access")
	ErrDownloadedLayerDiffSize = errors.New("size of downloaded layer is different from expected")
	stripFilesList             = []string{
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

func (p *Pvr) GetDockerManifest(image registry.Image, auth types.AuthConfig) (*schema2.Manifest, error) {
	r, err := p.GetDockerRegistry(image, auth)
	if err != nil {
		return nil, err
	}

	manifestV2, err := r.ManifestV2(context.Background(), image.Path, image.Reference())
	if err != nil {
		return nil, ErrImageNotFound
	}

	return &manifestV2, nil
}

func (p *Pvr) GetDockerConfig(manifestV2 *schema2.Manifest, image registry.Image, auth types.AuthConfig) (map[string]interface{}, error) {
	r, err := p.GetDockerRegistry(image, auth)
	if err != nil {
		return nil, err
	}

	blobsURL := fmt.Sprintf("%s/v2/%s/blobs/%s", r.URL, image.Path, manifestV2.Config.Digest)

	resp, err := http.Get(blobsURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		wwwHeaders := resp.Header["Www-Authenticate"][0]

		// Expected format: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"

		baseReg := `%s="(([a-z]|[A-Z]|[0-9]|\:|\/|\-|\_|\.)+)"`

		realmReg := regexp.MustCompile(fmt.Sprintf(baseReg, "realm"))
		realm := realmReg.FindStringSubmatch(wwwHeaders)[1]

		serviceReg := regexp.MustCompile(fmt.Sprintf(baseReg, "service"))
		service := serviceReg.FindStringSubmatch(wwwHeaders)[1]

		scopeReg := regexp.MustCompile(fmt.Sprintf(baseReg, "scope"))
		scope := scopeReg.FindStringSubmatch(wwwHeaders)[1]

		tokenURL := fmt.Sprintf("%s?service=%s&scope=%s", realm, service, scope)
		req, err := http.NewRequest(http.MethodGet, tokenURL, nil)
		if err != nil {
			return nil, err
		}

		if auth.Username != "" && auth.Password != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(auth.Username + ":" + auth.Password))
			req.Header.Set("Authorization", "Basic "+auth)
		}

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var tokenResponse map[string]interface{}
		err = json.Unmarshal(content, &tokenResponse)
		if err != nil {
			return nil, err
		}

		token := tokenResponse["token"].(string)
		req, err = http.NewRequest(http.MethodGet, blobsURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err = http.DefaultClient.Do(req)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrImageNotFound
	}

	if err != nil {
		return nil, err
	}

	blobContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var blob map[string]interface{}
	err = json.Unmarshal(blobContent, &blob)

	config := blob["config"].(map[string]interface{})

	return config, nil
}

// DownloadLayersFromLocalDocker : Download Layers From Local Docker
func DownloadLayersFromLocalDocker(imageID string) (io.ReadCloser, error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	cli.NegotiateAPIVersion(ctx)
	httpClient := cli.HTTPClient()
	url := "http://v" + cli.ClientVersion() + "/images/" + imageID + "/get"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res.Body, nil
}

// DockerImage : return type of ImageExistsInLocalDocker()
type DockerImage struct {
	Exists         bool
	ImageID        string
	DockerConfig   map[string]interface{}
	DockerManifest *schema2.Manifest
	DockerRegistry *registry.Registry
	ImagePath      string
}

// FindDockerImage : Find Docker Image
func (p *Pvr) FindDockerImage(app *AppData) error {
	app.LocalImage.Exists = false
	app.RemoteImage.Exists = false

	sourceOrder := strings.Split(app.Source, ",")
	for _, source := range sourceOrder {
		fmt.Printf("Checking repo in " + source + " docker\n")
		if source == "local" {
			err := LoadLocalImage(app)
			if err != nil {
				return err
			}
			if app.LocalImage.Exists {
				return nil
			}

		} else if source == "remote" {
			err := p.LoadRemoteImage(app)
			if err != nil {
				return err
			}
			if app.RemoteImage.Exists {
				return nil
			}
		} else {
			return errors.New("Invalid source:" + source)
		}
	}
	return errors.New("Image not found in source:" + app.Source + "\n")
}

// LoadRemoteImage : To check whether Image Exist In Remote Docker Or Not
func (p *Pvr) LoadRemoteImage(app *AppData) error {
	app.RemoteImage = DockerImage{
		Exists:  false,
		ImageID: "",
	}
	image, err := registry.ParseImage(app.From)
	if err != nil {
		return err
	}
	auth, err := p.AuthConfig(app.Username, app.Password, image.Domain)
	if err != nil {
		return err
	}
	dockerManifest, err := p.GetDockerManifest(image, auth)
	if err != nil {
		manifestErr := ReportDockerManifestError(err, app.From)
		if err.Error() == "image not found or you do not have access" {
			fmt.Printf(manifestErr.Error() + "\n")
			return nil
		}
		return manifestErr
	}
	dockerConfig, err := p.GetDockerConfig(dockerManifest, image, auth)
	if err != nil {
		err = ReportDockerManifestError(err, app.From)
		return err
	}
	dockerRegistry, err := p.GetDockerRegistry(image, auth)
	if err != nil {
		return err
	}
	app.Username = auth.Username
	app.Password = auth.Password

	app.RemoteImage.Exists = true
	app.RemoteImage.ImageID = string(dockerManifest.Config.Digest)
	app.RemoteImage.DockerConfig = dockerConfig
	app.RemoteImage.DockerManifest = dockerManifest
	app.RemoteImage.DockerRegistry = dockerRegistry
	app.RemoteImage.ImagePath = image.Path

	return nil
}

// LoadLocalImage : To check whether Image Exist In Local Docker Or Not
func LoadLocalImage(app *AppData) error {
	app.LocalImage = DockerImage{
		Exists:  false,
		ImageID: "",
	}
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	cli.NegotiateAPIVersion(ctx)
	defer cli.Close()
	if err != nil {
		return err
	}
	inspect, _, err := cli.ImageInspectWithRaw(ctx, app.From)
	if err != nil {
		if err.Error() == "Error: No such image: "+app.From {
			fmt.Printf("Repo not exists in local docker\n")
			return nil
		}
		return err
	}
	fmt.Printf("Repo exists in local docker\n")
	app.LocalImage.Exists = true
	app.LocalImage.ImageID = inspect.ID
	//Setting Docker Config values
	app.LocalImage.DockerConfig = map[string]interface{}{}
	app.LocalImage.DockerConfig["Hostname"] = inspect.Config.Hostname
	app.LocalImage.DockerConfig["Domainname"] = inspect.Config.Domainname
	app.LocalImage.DockerConfig["User"] = inspect.Config.User
	app.LocalImage.DockerConfig["AttachStdin"] = inspect.Config.AttachStdin
	app.LocalImage.DockerConfig["AttachStdout"] = inspect.Config.AttachStdout
	app.LocalImage.DockerConfig["AttachStderr"] = inspect.Config.AttachStderr
	app.LocalImage.DockerConfig["ExposedPorts"] = inspect.Config.ExposedPorts
	app.LocalImage.DockerConfig["Tty"] = inspect.Config.Tty
	app.LocalImage.DockerConfig["OpenStdin"] = inspect.Config.OpenStdin
	app.LocalImage.DockerConfig["StdinOnce"] = inspect.Config.StdinOnce
	app.LocalImage.DockerConfig["Env"] = inspect.Config.Env
	app.LocalImage.DockerConfig["Cmd"] = []interface{}{}
	for _, v := range inspect.Config.Cmd {
		app.LocalImage.DockerConfig["Cmd"] = append(app.LocalImage.DockerConfig["Cmd"].([]interface{}), v)
	}
	app.LocalImage.DockerConfig["Healthcheck"] = inspect.Config.Healthcheck
	app.LocalImage.DockerConfig["ArgsEscaped"] = inspect.Config.ArgsEscaped
	app.LocalImage.DockerConfig["Image"] = inspect.Config.Image
	app.LocalImage.DockerConfig["Volumes"] = inspect.Config.Volumes
	app.LocalImage.DockerConfig["WorkingDir"] = inspect.Config.WorkingDir
	app.LocalImage.DockerConfig["Entrypoint"] = inspect.Config.Entrypoint
	app.LocalImage.DockerConfig["NetworkDisabled"] = inspect.Config.NetworkDisabled
	app.LocalImage.DockerConfig["MacAddress"] = inspect.Config.MacAddress
	app.LocalImage.DockerConfig["OnBuild"] = inspect.Config.OnBuild
	app.LocalImage.DockerConfig["Labels"] = inspect.Config.Labels
	app.LocalImage.DockerConfig["StopSignal"] = inspect.Config.StopSignal
	app.LocalImage.DockerConfig["StopTimeout"] = inspect.Config.StopTimeout
	app.LocalImage.DockerConfig["Shell"] = inspect.Config.Shell
	return nil
}

// GetFileContentType : Get File Content Type of a file
func GetFileContentType(src string) (string, error) {
	file, err := os.Open(src)
	defer file.Close()
	if err != nil {
		return "", err
	}
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}
	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

// Untar : Untar a file or folder
func Untar(dst string, src string) error {
	contentType, err := GetFileContentType(src)
	if err != nil {
		return err
	}
	tarPath, err := exec.LookPath(TAR_CMD)
	if err != nil {
		return err
	}
	args := []string{tarPath, "xzvf", src, "-C", dst}
	if contentType == "application/octet-stream" {
		args = []string{tarPath, "xvf", src, "-C", dst}
	}
	untar := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	untar.Stdout = &out
	untar.Stderr = &stderr
	err = untar.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
	}
	return err
}

// AppData : To hold all required App Information
type AppData struct {
	Appname         string
	DockerURL       string
	Username        string
	Password        string
	Appmanifest     *Source
	TemplateArgs    map[string]interface{}
	DestinationPath string
	LocalImage      DockerImage
	RemoteImage     DockerImage
	From            string
	Source          string
	ConfigFile      string
	Volumes         []string
}

func (p *Pvr) GenerateApplicationSquashFS(app AppData) error {
	digestFile := filepath.Join(app.DestinationPath, DOCKER_DIGEST_FILE)
	digest := ""
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		digest = app.LocalImage.ImageID
	} else if app.RemoteImage.Exists {
		digest = app.RemoteImage.ImageID
	}

	currentDigest, err := os.Open(digestFile)
	if err == nil {
		currentDigestContent, err := ioutil.ReadAll(currentDigest)
		if err == nil {
			if string(currentDigestContent) == string(digest) {
				return nil
			}
		}
	}

	fmt.Println("Generating squashfs...")

	tempdir, err := ioutil.TempDir(os.TempDir(), "download-layer-")
	if err != nil {
		return err
	}

	defer os.RemoveAll(tempdir)

	files := []string{}
	fmt.Println("Downloading layers...")
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		fmt.Println("Downloading layers from local docker")
		imageReader, err := DownloadLayersFromLocalDocker(app.LocalImage.ImageID)
		if err != nil {
			return err
		}
		buf := bufio.NewReader(imageReader)
		filename := filepath.Join(tempdir, "layers") + ".tar.gz"
		file, err := os.Create(filename)
		if err != nil {
			return err
		}
		_, err = buf.WriteTo(file)
		if err != nil {
			return err
		}
		fmt.Printf("Layers downloaded from local\n")

		fmt.Printf("Extracting layers folder\n")

		os.MkdirAll(tempdir+"/layers", 0777)
		err = Untar(tempdir+"/layers", filename)
		if err != nil {
			return err
		}
		// Read layer.tar file locations from manifest.json
		manifestFile, err := ioutil.ReadFile(tempdir + "/layers/manifest.json")
		if err != nil {
			return err
		}
		manifestData := []map[string]interface{}{}
		err = json.Unmarshal([]byte(manifestFile), &manifestData)
		if err != nil {
			return err
		}
		//Populate layer.tar file locations in files array
		for _, layer := range manifestData[0]["Layers"].([]interface{}) {
			filename := filepath.Join(tempdir, "layers") + "/" + layer.(string)
			files = append(files, filename)
		}

	} else if app.RemoteImage.Exists {
		//Download from remote repo.
		for i, layer := range app.RemoteImage.DockerManifest.Layers {
			layerReader, err := app.RemoteImage.DockerRegistry.DownloadLayer(context.Background(), app.RemoteImage.ImagePath, layer.Digest)
			if err != nil {
				return err
			}

			buf := bufio.NewReader(layerReader)

			filename := filepath.Join(tempdir, strconv.Itoa(i)) + ".tar.gz"
			file, err := os.Create(filename)
			if err != nil {
				return err
			}

			writedCount, err := buf.WriteTo(file)
			if writedCount != layer.Size {
				return ErrDownloadedLayerDiffSize
			}

			if err != nil {
				return err
			}
			files = append(files, filename)
			fmt.Printf("Layer %d downloaded\n", i)
		}
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
		err := Untar(extractPath, file)
		if err != nil {
			return err
		}
		fmt.Printf("Extracting layer %d\n", layerNumber)
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

	tempSquashFile := filepath.Join(app.DestinationPath, SQUASH_FILE+".new")
	squashFile := filepath.Join(app.DestinationPath, SQUASH_FILE)

	squashExist, err := IsFileExists(squashFile)
	if err != nil {
		return err
	}
	// make sure the squashfs file did not exists
	if squashExist {
		err := os.Remove(squashFile)
		if err != nil {
			return err
		}
	}

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
	return ioutil.WriteFile(digestFile, []byte(digest), 0644)
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

func (p *Pvr) AuthConfig(username, password, registry string) (types.AuthConfig, error) {
	if registry == DOCKER_DOMAIN {
		auth, err := repoutils.GetAuthConfig(username, password, DOCKER_REGISTRY)
		auth.ServerAddress = DOCKER_REGISTRY_SERVER_ADDRESS
		return auth, err
	}

	return repoutils.GetAuthConfig(username, password, registry)
}

// IsFileExists : Check if File Exists  or not
func IsFileExists(filePath string) (bool, error) {
	//Check if file exists in cache
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}
