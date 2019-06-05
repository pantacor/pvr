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
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/genuinetools/reg/registry"
	"github.com/genuinetools/reg/repoutils"
	"github.com/urfave/cli"
)

const (
	pvrRegistry         = "registry.gitlab.com/pantacor/pvr"
	lastUpdateFile      = "installed_version"
	pvrCmd              = "pvr"
	cacheFolder         = "cache/"
	lastCheckedFileName = "last_checked"
	updateEveryDays     = 1.0
)

type downloadData struct {
	filename string
	number   int
	err      error
	cached   bool
}

type layerData struct {
	Registry  *registry.Registry
	Image     *registry.Image
	Layer     *distribution.Descriptor
	OutputDir *string
	Number    int
	Downloads chan<- *downloadData
}

// UpdateIfNecessary update pvr if is necesary but only check on time at the day
func UpdateIfNecessary(c *cli.Context) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	session, err := NewSession(c.App)

	if err != nil {
		return err
	}

	pvr, err := NewPvr(session, wd)
	if err != nil {
		return err
	}

	if !pvr.Session.Configuration.AutoUpgrade {
		return nil
	}

	lastCheckedPath := filepath.Join(session.GetConfigDir(), lastCheckedFileName)
	daysSinceLastModify, err := daysSinceLastUpdate(lastCheckedPath)
	if err != nil {
		return nil
	}

	if *daysSinceLastModify < updateEveryDays {
		return nil
	}

	username := c.String("username")
	password := c.String("password")
	pvr.UpdatePvr(&username, &password, true)
	WriteTxtFile(lastCheckedPath, time.Now().Format(time.RFC3339))

	return nil
}

// UpdatePvr Take the username, password and configuration File (aka: ~/.pvr) and update the pvr binary
func (pvr *Pvr) UpdatePvr(username, password *string, silent bool) error {
	currentDigest, previousDigest, manifestV2, err := pvr.getDigetsDifference(username, password)
	if err != nil {
		return err
	}

	if *currentDigest == *previousDigest {
		if silent != true {
			fmt.Println("You already have the latest version of PVR :) \n\r")
		}
		return nil
	}

	fmt.Printf("Starting update PVR using Docker latest tag (%v) \r\n ", *currentDigest)

	cacheFolder, err := pvr.downloadAdUpdateBinary(username, password, currentDigest, manifestV2)
	if err != nil {
		return err
	}

	fmt.Printf("\r\nDocker layers are going to be cache on: %v \r\n\r\n", *cacheFolder)

	fmt.Printf("PVR has been updated! \r\n\r\n ")
	return nil
}

func (pvr *Pvr) getDigetsDifference(username, password *string) (*string, *string, *schema2.Manifest, error) {
	configDir := pvr.Session.configDir
	tag := pvr.Session.Configuration.DistributionTag
	registry := pvrRegistry
	dockerURL := fmt.Sprintf("%s:%s", registry, tag)
	manifestV2, err := pvr.GetDockerManifest(dockerURL, *username, *password)
	if err != nil {
		return nil, nil, nil, err
	}

	currentDigest := string(manifestV2.Config.Digest)

	fileContent, err := ReadOrCreateFile(filepath.Join(configDir, lastUpdateFile))
	if err != nil {
		return nil, nil, nil, err
	}
	var previousDigest string
	if fileContent == nil {
		previousDigest = ""
	} else {
		previousDigest = string(*fileContent)
	}
	return &currentDigest, &previousDigest, manifestV2, nil
}

func (pvr *Pvr) downloadAdUpdateBinary(username, password, currentDigest *string, manifestV2 *schema2.Manifest) (*string, error) {
	configDir := pvr.Session.configDir
	cachePath := filepath.Join(configDir, cacheFolder)
	dockerURL := fmt.Sprintf("%s:%s", pvrRegistry, pvr.Session.Configuration.DistributionTag)

	err := CreateFolder(cachePath)
	if err != nil {
		return nil, err
	}

	temp, extractPath, err := pvr.getDockerContent(dockerURL, &cachePath, username, password, manifestV2)
	if err != nil {
		return nil, err
	}

	err = updatePvrBinary(extractPath)
	if err != nil {
		return nil, err
	}

	err = WriteTxtFile(filepath.Join(configDir, lastUpdateFile), *currentDigest)
	if err != nil {
		return nil, err
	}

	defer os.RemoveAll(*extractPath)

	return temp, nil
}

func (pvr *Pvr) getDockerContent(dockerURL string, outputDir, username, password *string, dockerManifest *schema2.Manifest) (*string, *string, error) {
	image, err := registry.ParseImage(dockerURL)
	if err != nil {
		return nil, nil, err
	}

	auth, err := repoutils.GetAuthConfig(*username, *password, image.Domain)
	if err != nil {
		return nil, nil, err
	}

	r, err := pvr.GetDockerRegistry(image, auth)
	if err != nil {
		return nil, nil, err
	}

	totalLayers := len(dockerManifest.Layers)
	downloads := make(chan *downloadData, totalLayers)
	var waitGroup sync.WaitGroup

	fmt.Printf("\n\rDownloading layers %d ... \r\n", totalLayers)

	waitGroup.Add(totalLayers)
	for i, layer := range dockerManifest.Layers {
		layerdata := layerData{
			Registry:  r,
			Image:     &image,
			Layer:     &layer,
			OutputDir: outputDir,
			Number:    i + 1,
			Downloads: downloads,
		}
		go func(layerdata *layerData) {
			downloadlayers(layerdata)
			waitGroup.Done()
		}(&layerdata)
	}

	go func() {
		waitGroup.Wait()
		close(downloads)
	}()

	files, err := processDownloads(downloads, totalLayers)

	extractPath := filepath.Join(*outputDir, "bin")
	os.MkdirAll(extractPath, 0777)

	fmt.Printf("\n\rExtracting layers %d ... \r\n", len(files))

	err = ExtractFiles(files, extractPath)
	if err != nil {
		return nil, nil, err
	}

	return outputDir, &extractPath, nil
}

func updatePvrBinary(extractPath *string) error {
	platform := GetPlatform()
	binLocation := filepath.Join(*extractPath, "/pkg/bin/", platform, pvrCmd)

	pvrPath, err := getExecutableFilePath()
	if err != nil {
		return err
	}

	err = os.Remove(*pvrPath)
	if err != nil {
		return err
	}

	err = os.Rename(binLocation, *pvrPath)
	if err != nil {
		return err
	}

	fmt.Printf("\r\nPvr installed on %v \r\n", *pvrPath)

	return nil
}

func processDownloads(downloads chan *downloadData, totalLayers int) ([]string, error) {
	fromMessages := map[bool]string{true: "from cache", false: "from repository"}
	files := []string{}

	for download := range downloads {
		if download.err != nil {
			return nil, download.err
		}
		fmt.Printf("Done with [%d/%d] %v \r\n", download.number, totalLayers, fromMessages[download.cached])
		files = append(files, download.filename)
	}
	return files, nil
}

func downloadlayers(layerdata *layerData) {
	i := layerdata.Number
	filename := filepath.Join(*layerdata.OutputDir, strconv.Itoa(i)) + ".tar.gz"

	sameFile, err := FileHasSameSha(filename, string(layerdata.Layer.Digest))
	if err != nil {
		layerdata.Downloads <- &downloadData{
			filename: filename,
			number:   i,
			err:      err,
		}
		return
	}

	if sameFile {
		layerdata.Downloads <- &downloadData{
			filename: filename,
			number:   i,
			err:      nil,
			cached:   true,
		}
		return
	}

	err = os.Remove(filename)
	if err != nil && !os.IsNotExist(err) {
		layerdata.Downloads <- &downloadData{
			filename: filename,
			number:   i,
			err:      err,
		}
		return
	}

	layerReader, err := layerdata.Registry.DownloadLayer(context.Background(), layerdata.Image.Path, layerdata.Layer.Digest)
	buf := bufio.NewReader(layerReader)

	file, err := os.Create(filename)
	if err != nil {
		layerdata.Downloads <- &downloadData{
			filename: filename,
			number:   i,
			err:      err,
		}
		return
	}
	_, err = buf.WriteTo(file)
	if err != nil {
		layerdata.Downloads <- &downloadData{
			filename: filename,
			number:   i,
			err:      err,
		}
		return
	}

	layerdata.Downloads <- &downloadData{
		filename: filename,
		number:   i,
		err:      nil,
		cached:   false,
	}
	return
}

func daysSinceLastUpdate(lastCheckedPath string) (*float64, error) {
	pvrPath, err := getExecutableFilePath()
	if err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(*pvrPath)
	if err != nil {
		return nil, err
	}

	lastCheckedFile, err := ReadOrCreateFile(lastCheckedPath)
	if err != nil {
		return nil, err
	}

	lastChecked := fileInfo.ModTime()

	if lastCheckedFile == nil {
		WriteTxtFile(lastCheckedPath, lastChecked.Format(time.RFC3339))
	} else {
		lastChecked, err = time.Parse(time.RFC3339, string(*lastCheckedFile))
		if err != nil {
			lastChecked = fileInfo.ModTime()
		}
	}

	daysSinceLastModify := time.Now().Sub(lastChecked).Hours() / 24

	return &daysSinceLastModify, nil
}

func getExecutableFilePath() (*string, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	defaultBinPath := filepath.Join(usr.HomeDir, "/bin/", pvrCmd)

	ex, err := os.Executable()
	if err != nil {
		return &defaultBinPath, nil
	}

	pvrPath, err := filepath.EvalSymlinks(ex)
	if err != nil {
		return &defaultBinPath, nil
	}

	return &pvrPath, nil
}
