//
// Copyright 2017-2023  Pantacor Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package libpvr

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
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
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"gitlab.com/pantacor/pvr/utils/pvjson"
)

const (
	MKDIR_CMD                      = "mkdir"
	RM_CMD                         = "rm"
	TAR_CMD                        = "tar"
	TOUCH_CMD                      = "touch"
	MAKE_SQUASHFS_CMD              = "mksquashfs"
	SQUASH_FILE                    = "root.squashfs"
	SQUASH_OVL_FILE                = "root.ovl.squashfs"
	DOCKER_DIGEST_SUFFIX           = ".docker-digest"
	ROOTFS_DIGEST_SUFFIX           = ".rootfs-digest"
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
	structureDirs = []string{
		"proc",
		"sys",
		"dev",
		"tmp",
		"var/tmp",
		"run",
		"var/run",
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

// GetDockerImageRepoDigest : Get Docker Image Repo Digest
func (p *Pvr) GetDockerImageRepoDigest(image registry.Image, auth types.AuthConfig) (string, error) {
	r, err := p.GetDockerRegistry(image, auth)
	if err != nil {
		return "", err
	}
	repoDigest, err := r.Digest(context.Background(), image)
	if err != nil {
		return "", err
	}
	return string(repoDigest), nil
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
		err = pvjson.Unmarshal(content, &tokenResponse)
		if err != nil {
			return nil, err
		}

		tokenData, ok := tokenResponse["token"]
		if !ok {
			tokenData, ok = tokenResponse["access_token"]
			if !ok {
				return nil, fmt.Errorf("can't get a access token for %s", tokenURL)
			}
		}

		token := tokenData.(string)
		req, err = http.NewRequest(http.MethodGet, blobsURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
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
	err = pvjson.Unmarshal(blobContent, &blob)
	if err != nil {
		return nil, err
	}

	config := blob["config"].(map[string]interface{})

	return config, nil
}

// DownloadLayersFromLocalDocker : Download Layers From Local Docker
func DownloadLayersFromLocalDocker(digest string) (io.ReadCloser, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	cli.NegotiateAPIVersion(ctx)
	httpClient := cli.HTTPClient()

	url := "http://v" + cli.ClientVersion() + "/images/" + digest + "/get"
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
	DockerDigest   string
	DockerConfig   map[string]interface{}
	DockerManifest *schema2.Manifest
	DockerRegistry *registry.Registry
	ImagePath      string
	DockerPlatform string
}

// FindDockerImage : Find Docker Image
func (p *Pvr) FindDockerImage(app *AppData) (err error) {
	app.LocalImage.Exists = false
	app.RemoteImage.Exists = false

	sourceOrder := strings.Split(app.Source, ",")
	for _, source := range sourceOrder {
		switch source {
		case "local":
			err = LoadLocalImage(app)
			if app.LocalImage.Exists {
				return nil
			}
		case "remote":
			err = p.LoadRemoteImage(app)
			if app.RemoteImage.Exists {
				return nil
			}
		default:
			return errors.New("source type not supported:" + source + "\n")
		}

		if err != nil {
			fmt.Printf("%s source had an error, trying with other sources \n %s \n", source, err)
		}
	}

	if err != nil {
		return err
	}

	return errors.New("Image not found in source:" + app.Source + "\n")
}

// LoadRemoteImage : To check whether Image Exist In Remote Docker Or Not
func (p *Pvr) LoadRemoteImage(app *AppData) error {

	var dockerManifest *schema2.Manifest

	app.RemoteImage = DockerImage{
		Exists: false,
	}
	image, err := registry.ParseImage(app.From)
	if err != nil {
		return err
	}
	auth, err := p.AuthConfig(app.Username, app.Password, image.Domain)
	if err != nil {
		return err
	}
	dockerRegistry, err := p.GetDockerRegistry(image, auth)
	if err != nil {
		return err
	}

	var repoDigest string
	var dockerPlatform string
	var platforms []interface{}

	if app.Platform == "" {
		dockerJsonI, ok := p.PristineJsonMap["_hostconfig/pvr/docker.json"]

		if ok {
			dockerJson := dockerJsonI.(map[string]interface{})
			platformsI, ok := dockerJson["platforms"]
			if ok {
				platforms = platformsI.([]interface{})
			}
		}
	} else {
		platforms = append(platforms, app.Platform)
	}

	// we go down the multiarch path if we have seen a platform
	// restriction in pvr-docker.json
	if platforms != nil {
		manifestList, err := dockerRegistry.ManifestList(context.Background(),
			image.Path, image.Reference())

		if err != nil {
			return err
		}

		for _, v := range manifestList.Manifests {
			for _, v1 := range platforms {
				v1S := v1.(string)
				p := strings.SplitN(v1S, "/", 3)
				if v.Platform.Architecture == p[1] &&
					(len(p) < 3 || p[2] == v.Platform.Variant) {
					repoDigest = v.Digest.String()
					dockerPlatform = v1S
					break
				}
			}
			if repoDigest != "" {
				dm, err := dockerRegistry.ManifestV2(context.Background(), image.Path, repoDigest)
				dockerManifest = &dm
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Found Manifest for platform %s\n",
					dockerPlatform)

				break
			} else {
				dockerPlatform = ""
			}
		}
	}

	if dockerManifest == nil {
		dockerManifest, err = p.GetDockerManifest(image, auth)
		if err != nil {
			manifestErr := ReportDockerManifestError(err, app.From)
			if err.Error() == "image not found or you do not have access" {
				fmt.Fprintf(os.Stderr, manifestErr.Error()+"\n")
				return nil
			}
			return manifestErr
		}
		fmt.Fprintf(os.Stderr, "Found Manifest for default platform.\n")
	}

	dockerConfig, err := p.GetDockerConfig(dockerManifest, image, auth)
	if err != nil {
		err = ReportDockerManifestError(err, app.From)
		return err
	}

	// if we cannot find our arch we go the old direct way of retrieving repo
	if repoDigest == "" && app.RemoteImage.DockerPlatform != "" {
		return errors.New("no docker image found for platform " + app.RemoteImage.DockerPlatform)
	} else if repoDigest == "" {
		repoDigest, err = p.GetDockerImageRepoDigest(image, auth)
		if err != nil {
			return err
		}
	}

	li := strings.LastIndex(app.From, ":")
	var imageName string
	if li < 0 {
		imageName = app.From
	} else {
		splits := []string{app.From[:li], app.From[li+1:]}
		imageName = splits[0]
	}

	//Extract image name from repo digest. eg: Extract "busybox" from "busybox@sha256:afe605d272837ce1732f390966166c2afff5391208ddd57de10942748694049d"
	if strings.Contains(imageName, "@sha256") {
		splits := strings.Split(imageName, "@")
		imageName = splits[0]
	}

	if !strings.Contains(repoDigest, "@") {
		repoDigest = imageName + "@" + repoDigest
	}

	app.Username = auth.Username
	app.Password = auth.Password

	app.RemoteImage.Exists = true
	app.RemoteImage.DockerDigest = repoDigest
	app.RemoteImage.DockerConfig = dockerConfig
	app.RemoteImage.DockerManifest = dockerManifest
	app.RemoteImage.DockerRegistry = dockerRegistry
	app.RemoteImage.DockerPlatform = dockerPlatform
	app.RemoteImage.ImagePath = image.Path

	return nil
}

// LoadLocalImage : To check whether Image Exist In Local Docker Or Not
func LoadLocalImage(app *AppData) error {
	app.LocalImage = DockerImage{
		Exists: false,
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
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
	if len(inspect.RepoDigests) > 0 {
		app.LocalImage.DockerDigest = inspect.RepoDigests[0]
	} else {
		app.LocalImage.DockerDigest = inspect.ID
	}

	//Setting Docker Config values
	configData, err := json.Marshal(inspect.Config)
	if err != nil {
		return err
	}
	err = pvjson.Unmarshal(configData, &app.LocalImage.DockerConfig)
	if err != nil {
		return err
	}
	return nil
}

// AppData : To hold all required App Information
type AppData struct {
	SquashFile      string
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
	Platform        string
	ConfigFile      string
	Volumes         []string
	FormatOptions   string
	SourceType      string
	DoOverlay       bool
	Base            string
}

func (p *Pvr) GenerateApplicationSquashFS(app *AppData, appManifest *Source) error {
	digestFile := filepath.Join(app.DestinationPath, app.SquashFile+DOCKER_DIGEST_SUFFIX)
	digest := ""
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		digest = app.LocalImage.DockerDigest
	} else if app.RemoteImage.Exists {
		digest = app.RemoteImage.DockerDigest
	}

	currentDigest, err := os.Open(digestFile)
	if err == nil {
		defer currentDigest.Close()
		currentDigestContent, err := ioutil.ReadAll(currentDigest)
		if err == nil {
			if string(currentDigestContent) == string(digest) && appManifest.Base == "" {
				return nil
			}
		}
	}

	configDir := p.Session.configDir
	cacheDir := filepath.Join(configDir, cacheFolder)
	err = CreateFolder(cacheDir)

	if err != nil {
		return fmt.Errorf("couldn't create cache folder %v", err)
	}

	tempdir, err := ioutil.TempDir(os.TempDir(), "download-layer-")
	if err != nil {
		return err
	}

	defer RemoveAll(tempdir)

	files := []string{}
	fmt.Println("Downloading layers...")
	//	Exists flag is true only if the image got loaded which will depend on
	//  priority order provided in --source=local,remote
	if app.LocalImage.Exists {
		filename := filepath.Join(cacheDir, app.LocalImage.DockerDigest) + ".tar.gz"
		fileExistInCache, err := IsFileExists(filename)
		if err != nil {
			return err
		}
		if fileExistInCache {
			fmt.Println("Layers Found in Cache")
			fmt.Printf("Extracting layers folder(cache)\n")
		} else {
			fmt.Println("Layers Not Found in Cache")
			fmt.Println("Downloading layers from local docker")
			imageReader, err := DownloadLayersFromLocalDocker(app.LocalImage.DockerDigest)
			if err != nil {
				return err
			}
			buf := bufio.NewReader(imageReader)
			//filename := filepath.Join(tempdir, "layers") + ".tar.gz"
			file, err := os.Create(filename)
			if err != nil {
				return err
			}
			_, err = buf.WriteTo(file)
			if err != nil {
				return err
			}
			fmt.Printf("Layers downloaded from local docker\n")
			fmt.Printf("Extracting layers folder\n")
		}

		MkdirAll(tempdir+"/layers", 0777)
		err = Untar(tempdir+"/layers", filename, []string{"--exclude", ".wh.*"})
		if err != nil {
			return err
		}
		// Read layer.tar file locations from manifest.json
		manifestFile, err := ioutil.ReadFile(tempdir + "/layers/manifest.json")
		if err != nil {
			return err
		}
		manifestData := []map[string]interface{}{}
		err = pvjson.Unmarshal([]byte(manifestFile), &manifestData)
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
			filename := filepath.Join(cacheDir, string(layer.Digest)) + ".tar.gz"
			shaValid, err := FileHasSameSha(filename, string(layer.Digest))
			if err != nil {
				return err
			}
			if shaValid {
				fmt.Printf("Layer %d downloaded(cache)\n", i)
				files = append(files, filename)
				continue
			}

			layerReader, err := app.RemoteImage.DockerRegistry.DownloadLayer(context.Background(), app.RemoteImage.ImagePath, layer.Digest)
			if err != nil {
				return err
			}

			buf := bufio.NewReader(layerReader)

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
	MkdirAll(extractPath, 0777)
	defer os.RemoveAll(extractPath)

	tarPath, err := exec.LookPath(TAR_CMD)
	if err != nil {
		return err
	}

	if tarPath == "" {
		return ErrTarNotFound
	}

	fmt.Println("Extracting layers...")
	for layerNumber, file := range files {
		err := ProcessWhiteouts(extractPath, file, layerNumber)
		if err != nil {
			log.Println("Error processing whiteouts.")
			return err
		}
		err = Untar(extractPath, file, []string{"--exclude", ".wh.*"})
		if err != nil {
			return err
		}
		fmt.Printf("Extracting layer %d\n", layerNumber)
	}

	fmt.Println("Stripping qemu files...")
	for _, file := range stripFilesList {
		fileToDelete := filepath.Join(extractPath, file)
		Remove(fileToDelete)
		if err != nil {
			return err
		}

		PrintDebugf("Deleted %s file\n", fileToDelete)
	}

	fmt.Println("Adding essential structural dirs to operate RO containers")
	for _, file := range structureDirs {
		dirToMake := filepath.Join(extractPath, file)
		os.MkdirAll(dirToMake, 0755)
	}

	// if we are using a different name, we generate an overlay
	if app.SquashFile == SQUASH_OVL_FILE {
		basePath, err := os.MkdirTemp("", "base-layer-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(basePath)
		ovlPath, err := os.MkdirTemp("", "ovl-*")
		if err != nil {
			return err
		}
		from := path.Join(p.Dir, app.Appname, SQUASH_FILE)
		if appManifest.Base != "" {
			from = path.Join(p.Dir, appManifest.Base, SQUASH_FILE)
		}
		cmd := exec.Command("unsquashfs", "-f", "-d", basePath, from)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return err
		}
		treeDiff := MkTreeDiff(basePath, extractPath)
		treeDiff.MkOvl(ovlPath)
		defer os.RemoveAll(ovlPath)
		extractPath = ovlPath
	} else if app.SquashFile != SQUASH_FILE {
		return errors.New("Unsupported SquashFile: " + app.SquashFile + ". Supported: " + SQUASH_FILE + ", " + SQUASH_OVL_FILE)
	}

	err = MakeSquash(extractPath, app)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(digestFile, []byte(digest), 0644)
}

// ProcessWhiteouts : FInd Whiteouts from a layer and process it in a given extract path
func ProcessWhiteouts(extractPath string, layerPath string, layerNumber int) error {
	whiteouts, err := FindWhiteoutsFromLayer(layerPath)
	if err != nil {
		return err
	}
	if len(whiteouts) == 0 {
		return nil
	}
	PrintDebugf("Processing Whiteouts from layer %d:%s\n", layerNumber, layerPath)
	for _, whiteoutFile := range whiteouts {

		basename := filepath.Base(whiteoutFile)
		dir := filepath.Join(extractPath, filepath.Dir(whiteoutFile))

		if strings.HasPrefix(basename, ".wh.") && strings.HasSuffix(basename, "opq") {
			//Clear all contents of the folder from the extract path
			PrintDebugln("Processing 'opq' whiteout for:" + dir)
			err := RemoveDirContents(dir)
			if err != nil && !os.IsNotExist(err) {
				fmt.Printf("WARNING: cannot process 'opq' whiteout %s (err=%s)\n", whiteoutFile, err.Error())
				continue
			}

		} else if strings.HasPrefix(basename, ".wh.") {
			//Remove the indicated file from the extract path'
			filePath := filepath.Join(dir, strings.TrimPrefix(basename, ".wh."))
			PrintDebugln("Processing whiteout:" + filePath)
			err := RemoveAll(filePath) //removr a file / dir
			if err != nil && !os.IsNotExist(err) {
				fmt.Printf("WARNING: cannot process whiteout %s (err=%s)\n", whiteoutFile, err.Error())
				continue
			}
		}
	}
	return nil
}

// FindWhiteoutsFromLayer : Find Whiteout Files From a Layer
func FindWhiteoutsFromLayer(layerPath string) ([]string, error) {
	whiteoutPaths := []string{}
	tarFile, err := os.Open(layerPath)
	if err != nil {
		return whiteoutPaths, err
	}
	contentType, err := GetFileContentType(layerPath)
	if err != nil {
		return whiteoutPaths, err
	}

	tr := tar.NewReader(tarFile)

	if contentType == "application/x-gzip" {
		// For zip content types, eg: application/x-gzip
		gzr, err := gzip.NewReader(tarFile)
		if err != nil {
			return whiteoutPaths, err
		}
		defer gzr.Close()
		tr = tar.NewReader(gzr)
	}

	for {
		path, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return whiteoutPaths, err
		}
		basename := filepath.Base(path.Name)
		if strings.HasPrefix(basename, ".wh.") {
			if !SliceContainsItem(whiteoutPaths, path.Name) {
				whiteoutPaths = append(whiteoutPaths, path.Name)
			}
		}
	}
	return whiteoutPaths, nil
}
func (p *Pvr) GetSquashFSDigest(squashFile, appName string) (string, error) {
	content, err := ioutil.ReadFile(filepath.Join(p.Dir, appName, squashFile+DOCKER_DIGEST_SUFFIX))
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
