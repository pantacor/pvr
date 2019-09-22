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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty"
)

// GetLocalDevice : Get Local Device updates
func (pvr *Pvr) GetLocalDevice(deviceURL string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	apiURL, err := pvr.Session.GetLocalDeviceAPIURL(deviceURL, "GET")
	if err != nil {
		return err
	}
	filename, err := DownloadFile(apiURL, wd)
	if err != nil {
		return err
	}
	//unpack tarball
	err = Untar(wd+"/.pvr", filename)
	if err != nil {
		return err
	}
	//removing tar file
	err = os.Remove(filename)
	if err != nil {
		return err
	}
	return nil
}

// CloneLocalDevice : Clone Local Device
func (s *Session) CloneLocalDevice(deviceURL, deviceDir string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	apiURL, err := s.GetLocalDeviceAPIURL(deviceURL, "CLONE")
	if err != nil {
		return err
	}
	filename, err := DownloadFile(apiURL, wd)
	if err != nil {
		return err
	}
	u, err := url.Parse(apiURL)
	if err != nil {
		return err
	}
	if deviceDir == "" {
		deviceDir = u.Hostname()
	}
	//Make device root directory
	pvrDir := wd + "/" + deviceDir + "/.pvr"
	err = CreateFolder(pvrDir)
	if err != nil {
		return err
	}
	//unpack tarball
	err = Untar(pvrDir, filename)
	if err != nil {
		return err
	}
	//removing tar file
	err = os.Remove(filename)
	if err != nil {
		return err
	}
	//pvr checkout
	err = s.CheckoutDevice(deviceDir)
	if err != nil {
		return err
	}
	//Saving Device IP to .pvr/config
	s.SaveLocalDeviceIP(deviceURL, deviceDir)
	return nil
}

// GetLocalDeviceAPIURL : Get Local Device API URL from host string
func (s *Session) GetLocalDeviceAPIURL(host string, api string) (string, error) {
	if !strings.HasPrefix(host, "http") &&
		!strings.HasPrefix(host, "https") {
		host = "http://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return "", err
	}
	port := u.Port()
	if port == "" {
		port = "12356"
	}
	revision := strings.Replace(u.RequestURI(), "/", "", 1)
	if revision != "" {
		revision = "?revision=" + revision
	}
	url := ""
	if api == "CLONE" || api == "GET" {
		url = u.Scheme + "://" + u.Hostname() + ":" + port + "/cgi-bin/pvrlocal" + revision
	} else if api == "POST" {
		url = u.Scheme + "://" + u.Hostname() + ":" + port + "/cgi-bin/pvrlocal"
	} else if api == "LOGS" {
		url = u.Scheme + "://" + u.Hostname() + ":" + port + "/cgi-bin/logs"
	}
	return url, nil
}

//SaveLocalDeviceIP : Save Local Device IP
func (s *Session) SaveLocalDeviceIP(deviceURL, deviceDir string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	pvr, err := NewPvr(s, wd+"/"+deviceDir)
	if err != nil {
		return err
	}
	u, err := url.Parse(deviceURL)
	if err != nil {
		return err
	}
	pvr.Pvrconfig.DefaultLocalDeviceURL = u.Scheme + "://" + u.Hostname() + ":" + u.Port()
	err = pvr.SaveConfig()
	if err != nil {
		return err
	}
	return nil
}

// PostToLocalDevice : to post to a local device
func (pvr *Pvr) PostToLocalDevice(deviceURL string) error {
	//Get API URL of local device
	apiURL, err := pvr.Session.GetLocalDeviceAPIURL(deviceURL, "POST")
	if err != nil {
		return err
	}
	//Generate a tar file
	tempdir, err := ioutil.TempDir(os.TempDir(), "device-tar-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempdir)
	os.MkdirAll(tempdir, 0777)
	SetTempFilesInterrupHandler(tempdir) //handling interruption
	tarFilePath := tempdir + "/device.tgz"
	err = pvr.Export(tarFilePath)
	if err != nil {
		return err
	}
	//get sha of tar file
	sha, err := FiletoSha(tarFilePath)
	if err != nil {
		return err
	}
	reader, err := os.Open(tarFilePath)
	if err != nil {
		return err
	}
	counter := &UploadCounter{}
	apiURL = apiURL + "?sha=" + sha
	res, err := resty.R().
		SetBody(io.TeeReader(reader, counter)).
		Post(apiURL)
	if err != nil {
		return err
	}
	//removing tar file
	err = os.Remove(tarFilePath)
	if err != nil {
		return err
	}
	if res.StatusCode() == http.StatusOK {
		return nil
	}
	fmt.Println("\nError posting to local device:" + apiURL + "\n")
	return errors.New("Error posting to local device:" + apiURL)
}

// GetLocalDeviceLogs : to get Local device logs
func (pvr *Pvr) GetLocalDeviceLogs(deviceURL string) error {
	apiURL, err := pvr.Session.GetLocalDeviceAPIURL(deviceURL, "LOGS")
	if err != nil {
		return err
	}
	startTime := time.Now().Add(time.Duration(-1 * time.Minute))
	tail := "100"
	urlParams := url.Values{}
	urlParams.Set("startdate", startTime.String())
	urlParams.Set("tail", tail)
	apiURL = apiURL + "?" + urlParams.Encode()
	res, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(res.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		fmt.Print(string(line))
	}
	return nil
}
