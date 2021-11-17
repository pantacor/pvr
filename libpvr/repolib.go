//
// Copyright 2017-2021  Pantacor Ltd.
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
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	jsonpatch "github.com/asac/json-patch"
	"github.com/cavaliercoder/grab"
	cjson "github.com/gibson042/canonicaljson-go"
	"github.com/go-resty/resty"
	"gitlab.com/pantacor/pantahub-base/objects"
	pvrapi "gitlab.com/pantacor/pvr/api"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/cheggaaa/pb.v1"
)

const PvrCheckpointFilename = "checkpoint.json"

type PvrStatus struct {
	NewFiles       []string
	RemovedFiles   []string
	ChangedFiles   []string
	UntrackedFiles []string
	JsonDiff       *[]byte
}

// stringify of file status for "pvr status" list...
func (p *PvrStatus) String() string {
	str := ""
	for _, f := range p.NewFiles {
		str += "A " + f + "\n"
	}
	for _, f := range p.RemovedFiles {
		str += "D " + f + "\n"
	}
	for _, f := range p.ChangedFiles {
		str += "C " + f + "\n"
	}
	for _, f := range p.UntrackedFiles {
		str += "? " + f + "\n"
	}
	return str
}

type PvrMap map[string]interface{}
type PvrIndex map[string]string

type Pvr struct {
	Initialized     bool
	Dir             string
	Pvrdir          string
	Pvdir           string
	Objdir          string
	Pvrconfig       PvrConfig
	PristineJson    []byte
	PristineJsonMap PvrMap
	NewFiles        PvrIndex
	Session         *Session
}

type PvrConfig struct {
	DefaultGetUrl  string
	DefaultPutUrl  string
	DefaultPostUrl string
	ObjectsDir     string

	// tokens by realm
	AccessTokens  map[string]string
	RefreshTokens map[string]string
}

type WrappableRestyCallFunc func(req *resty.Request) (*resty.Response, error)

func (p *Pvr) String() string {
	return "PVR: " + p.Dir
}

func NewPvr(s *Session, dir string) (*Pvr, error) {
	return NewPvrInit(s, dir)
}

func NewPvrInit(s *Session, dir string) (*Pvr, error) {
	pvr := Pvr{
		Dir:         dir + string(filepath.Separator),
		Pvrdir:      filepath.Join(dir, ".pvr"),
		Pvdir:       filepath.Join(dir, ".pv"),
		Objdir:      filepath.Join(dir, ".pvr", "objects"),
		Initialized: false,
		Session:     s,
	}

	pvr.Pvrconfig.AccessTokens = make(map[string]string)
	pvr.Pvrconfig.RefreshTokens = make(map[string]string)

	fileInfo, err := os.Stat(pvr.Dir)

	if os.IsNotExist(err) {
		err = os.MkdirAll(pvr.Dir, 0700)
		if err != nil {
			return nil, err
		}
		fileInfo, err = os.Stat(pvr.Dir)
	}

	if err != nil {
		return nil, err
	}

	if !fileInfo.IsDir() {
		return nil, errors.New("pvr path is not a directory: " + dir)
	}

	fileInfo, err = os.Stat(filepath.Join(pvr.Pvrdir, "json"))

	if err == nil && fileInfo.IsDir() {
		return nil, errors.New("Repo is in bad state. .pvr/json is a directory")
	}

	if err != nil {
		pvr.Initialized = false
		return &pvr, nil
	}

	jPath := filepath.Join(pvr.Pvrdir, "json")
	_, err = os.Stat(jPath)
	// pristine json we keep as string as this will allow users load into
	// convenient structs
	if err == nil {
		byteJSON, err := ioutil.ReadFile(jPath)
		if err != nil {
			return nil, err
		}
		pvr.PristineJson = byteJSON
	} else {
		fmt.Fprintln(os.Stderr, "WARN: pvr location ("+jPath+") is not a pvr repository; filling the gaps...")
		pvr.PristineJson = []byte("{}")
	}

	err = json.Unmarshal(pvr.PristineJson, &pvr.PristineJsonMap)
	if err != nil {
		return nil, errors.New("JSON Unmarshal (" + strings.TrimPrefix(filepath.Join(pvr.Pvrdir, "json"), pvr.Dir) + "): " + err.Error())
	}

	// new files is a json file we will parse happily
	bytesNew, err := ioutil.ReadFile(filepath.Join(pvr.Pvrdir, "new"))
	if err == nil {
		err = json.Unmarshal(bytesNew, &pvr.NewFiles)
	} else {
		pvr.NewFiles = map[string]string{}
		err = nil
	}

	if err != nil {
		return &pvr, errors.New("Repo in bad state. JSON Unmarshal (" + strings.TrimPrefix(filepath.Join(pvr.Pvrdir, "json"), pvr.Dir) + ") Not possible. Make a copy of the repository for forensics, file a bug and maybe delete that file manually to try to recover: " + err.Error())
	}

	fileInfo, err = os.Stat(filepath.Join(pvr.Pvrdir, "config"))

	if err == nil && fileInfo.IsDir() {
		return nil, errors.New("Repo is in bad state. .pvr/json is a directory")
	} else if err == nil {
		byteJson, err := ioutil.ReadFile(filepath.Join(pvr.Pvrdir, "config"))

		err = json.Unmarshal(byteJson, &pvr.Pvrconfig)
		if err != nil {
			return nil, errors.New("JSON Unmarshal (" + strings.TrimPrefix(filepath.Join(pvr.Pvrdir, "json"), pvr.Dir) + "): " + err.Error())
		}
	} else {
		// not exist
	}

	if pvr.Pvrconfig.ObjectsDir != "" {
		pvr.Objdir = pvr.Pvrconfig.ObjectsDir
		if !filepath.IsAbs(pvr.Objdir) {
			pvr.Objdir = path.Join(pvr.Pvrdir, "..", pvr.Objdir)
		}
	}
	return &pvr, nil
}

func (p *Pvr) addPvrFile(path string) error {
	shaBal, err := FiletoSha(path)
	if err != nil {
		return err
	}
	relPath := strings.TrimPrefix(path, p.Dir)
	relPathSlash := filepath.ToSlash(relPath)
	if p.NewFiles == nil {
		p.NewFiles = map[string]string{}
	}
	p.NewFiles[relPathSlash] = shaBal
	return nil
}

// XXX: make this git style
func (p *Pvr) AddFile(globs []string) error {

	err := filepath.Walk(p.Dir, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(walkPath, p.Pvrdir) || strings.HasPrefix(walkPath, p.Pvdir) {
			return nil
		}

		// no globs specified: add all
		if len(globs) == 0 || (len(globs) == 1 && globs[0] == ".") {
			p.addPvrFile(walkPath)
		}
		for _, glob := range globs {
			absglob := glob

			if !filepath.IsAbs(absglob) && absglob[0] != '/' {
				absglob = p.Dir + glob
			}
			matched, err := filepath.Match(absglob, walkPath)
			if err != nil {
				fmt.Fprintln(os.Stderr, "WARNING: cannot read file ("+err.Error()+"):"+walkPath)
				return err
			}
			if matched {
				err = p.addPvrFile(walkPath)
				if err != nil {
					return nil
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	jsonData, err := cjson.Marshal(p.NewFiles)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(p.Pvrdir, "new.XXX"), jsonData, 0644)
	if err != nil {
		return err
	}
	err = os.Rename(filepath.Join(p.Pvrdir, "new.XXX"), filepath.Join(p.Pvrdir, "new"))
	if err != nil {
		return err
	}
	return nil
}

// create the canonical json for the working directory
func (p *Pvr) GetWorkingJson() ([]byte, []string, error) {

	untrackedFiles := []string{}
	workingJson := map[string]interface{}{}
	currentSpec, ok := p.PristineJsonMap["#spec"]
	if ok {
		currentSpecString := currentSpec.(string)
		workingJson["#spec"] = currentSpecString
	} else {
		workingJson["#spec"] = "pantavisor-service-system@1"
	}

	err := filepath.Walk(p.Dir, func(filePath string, info os.FileInfo, err error) error {
		relPath := strings.TrimPrefix(filePath, p.Dir)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPathSlash := filepath.ToSlash(relPath)
		// ignore .pvr and .pv directories
		if _, ok := p.PristineJsonMap[relPathSlash]; !ok {
			if _, ok1 := p.NewFiles[relPathSlash]; !ok1 {
				if strings.HasPrefix(relPathSlash, ".pvr/") {
					return nil
				}
				if strings.HasPrefix(relPathSlash, ".pv/") {
					return nil
				}
				untrackedFiles = append(untrackedFiles, relPath)
				return nil
			}
		}
		// inline json
		if strings.HasSuffix(filepath.Base(filePath), ".json") {
			jsonFile := map[string]interface{}{}

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return err
			}

			err = json.Unmarshal(data, &jsonFile)
			if err != nil {
				return errors.New("JSON Unmarshal (" + strings.TrimPrefix(filePath, p.Dir) + "): " + err.Error())
			}
			workingJson[relPathSlash] = jsonFile
		} else {
			sha, err := FiletoSha(filePath)
			if err != nil {
				return err
			}
			workingJson[relPathSlash] = sha
		}

		return nil
	})

	if err != nil {
		return []byte{}, []string{}, err
	}

	b, err := cjson.Marshal(workingJson)

	if err != nil {
		return []byte{}, []string{}, err
	}

	return b, untrackedFiles, err
}

func (p *Pvr) Init(objectsDir string) error {

	return p.InitCustom("", objectsDir)
}

func (p *Pvr) InitCustom(customInitJson string, objectsDir string) error {

	var EMPTY_PVR_JSON string = `
{
	"#spec": "pantavisor-multi-platform@1"
}`

	_, err := os.Stat(p.Pvrdir)

	if err == nil {
		return errors.New("pvr init: .pvr directory/file found (" + p.Pvrdir + "). Cannot initialize an existing repository.")
	}

	err = os.Mkdir(p.Pvrdir, 0755)
	if err != nil {
		return err
	}

	// allow overwrite and remember abs path in config
	if objectsDir != "" {
		p.Objdir, err = filepath.Abs(objectsDir)

		if err != nil {
			return errors.New("Unexpected Error 1: " + err.Error())
		}

		p.Pvrconfig.ObjectsDir = p.Objdir
		p.SaveConfig()
	}

	err = os.Mkdir(p.Objdir, 0755)

	jsonFile, err := os.OpenFile(filepath.Join(p.Pvrdir, "json"), os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	if customInitJson != "" {
		_, err = jsonFile.Write([]byte(customInitJson))
	} else {
		_, err = jsonFile.Write([]byte(EMPTY_PVR_JSON))
	}

	return err
}

func (p *Pvr) Diff() (*[]byte, error) {
	workingJson, _, err := p.GetWorkingJson()
	if err != nil {
		return nil, err
	}

	diff, err := jsonpatch.CreateMergePatch(p.PristineJson, workingJson)

	if err != nil {
		return nil, err
	}

	return &diff, nil
}

func (p *Pvr) Status() (*PvrStatus, error) {
	rs := PvrStatus{}

	workingJson, untrackedFiles, err := p.GetWorkingJson()
	if err != nil {
		return nil, err
	}

	diff, err := jsonpatch.CreateMergePatch(p.PristineJson, workingJson)
	if err != nil {
		return nil, err
	}

	rs.JsonDiff = &diff

	// make json map out of diff
	diffJson := map[string]interface{}{}
	err = json.Unmarshal(*rs.JsonDiff, &diffJson)
	if err != nil {
		return nil, err
	}

	// run
	for file := range diffJson {
		val := diffJson[file]

		if val == nil {
			rs.RemovedFiles = append(rs.RemovedFiles, file)
			continue
		}

		// if we have this key in pristine, then file was changed
		if _, ok := p.PristineJsonMap[file]; ok {
			rs.ChangedFiles = append(rs.ChangedFiles, file)
			continue
		} else {
			rs.NewFiles = append(rs.NewFiles, file)
		}
	}

	rs.UntrackedFiles = untrackedFiles

	return &rs, nil
}

func (p *Pvr) prepCommitCheckpoint() error {

	fPath := path.Join(p.Dir, PvrCheckpointFilename)
	fNewPath := fPath + ".new"
	var fd *os.File

	p.NewFiles[PvrCheckpointFilename] = ""
	checkpointInfo := map[string]interface{}{
		"major": time.Now().Format(time.RFC3339),
	}
	buf, err := cjson.Marshal(checkpointInfo)
	if err != nil {
		goto exit
	}

	fd, err = os.OpenFile(fNewPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		goto exit
	}
	defer fd.Close()

	_, err = fd.Write(buf)
	if err != nil {
		goto exit
	}

	fd.Close()

	err = os.Rename(fNewPath, fPath)

	if err != nil {
		goto exit
	}

	// lets remember this file as NEW file right away ...
	if p.NewFiles[PvrCheckpointFilename] == "" {
		p.NewFiles[PvrCheckpointFilename] = "NO SHA NEEDED FOR JSON"
	}

exit:
	os.Remove(fNewPath)
	return err
}

func (p *Pvr) Commit(msg string, isCheckpoint bool) (err error) {

	// lets generate checkpoint file
	if isCheckpoint {
		err = p.prepCommitCheckpoint()
	}

	status, err := p.Status()

	if err != nil {
		return err
	}

	for _, v := range status.ChangedFiles {
		fmt.Fprintln(os.Stderr, "Committing "+filepath.Join(p.Objdir, v))
		if strings.HasSuffix(v, ".json") {
			continue
		}
		sha, err := FiletoSha(v)
		if err != nil {
			return err
		}
		err = Copy(filepath.Join(p.Objdir, sha), v)
		if err != nil {
			return err
		}
	}

	// copy all objects with atomic commit
	for _, v := range status.NewFiles {
		fmt.Fprintln(os.Stderr, "Adding "+v)
		if strings.HasSuffix(v, ".json") {
			continue
		}
		v = filepath.Join(p.Dir, v)
		sha, err := FiletoSha(v)
		if err != nil {
			return err
		}
		_, err = os.Stat(filepath.Join(p.Objdir, sha))
		// if not exists, then copy; otherwise continue
		if err != nil {

			err = Copy(filepath.Join(p.Objdir, sha+".new"), v)
			if err != nil {
				return err
			}
			err = os.Rename(filepath.Join(p.Objdir, sha+".new"),
				path.Join(p.Objdir, sha))
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}

	for _, v := range status.RemovedFiles {
		fmt.Fprintln(os.Stderr, "Removing "+v)
	}

	ioutil.WriteFile(filepath.Join(p.Pvrdir, "commitmsg.new"), []byte(msg), 0644)
	err = os.Rename(filepath.Join(p.Pvrdir, "commitmsg.new"), filepath.Join(p.Pvrdir, "commitmsg"))
	if err != nil {
		return err
	}

	newJson, err := jsonpatch.MergePatch(p.PristineJson, *status.JsonDiff)

	if err != nil {
		return err
	}

	newJson, err = FormatJsonC(newJson)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(p.Pvrdir, "json.new"), newJson, 0644)

	if err != nil {
		return err
	}

	err = os.Rename(filepath.Join(p.Pvrdir, "json.new"), filepath.Join(p.Pvrdir, "json"))

	if err != nil {
		return err
	}

	// ignore error here as new might not exist
	os.Remove(filepath.Join(p.Pvrdir, "new"))

	return err
}

func (p *Pvr) PutLocal(repoPath string) error {

	_, err := os.Stat(repoPath)
	if err != os.ErrNotExist {
		err = os.MkdirAll(repoPath, 0755)
	}
	if err != nil {
		return err
	}

	tP, err := NewPvr(p.Session, repoPath)
	objectsPath := tP.Pvrconfig.ObjectsDir

	info, err := os.Stat(objectsPath)
	if err == nil && !info.IsDir() {
		return errors.New("PVR repo directory in inusable state (objects is not a directory)")
	} else if err != nil {
		err = os.MkdirAll(objectsPath, 0755)
	}
	if err != nil {
		return err
	}

	// push all objects
	for k := range p.PristineJsonMap {
		if strings.HasSuffix(k, ".json") {
			continue
		}
		v := p.PristineJsonMap[k].(string)
		Copy(filepath.Join(objectsPath, v)+".new", filepath.Join(p.Dir, ".pvr", v))

	}
	err = filepath.Walk(p.Objdir, func(filePath string, info os.FileInfo, err error) error {
		// ignore directories
		if info.IsDir() {
			return nil
		}
		base := path.Base(filePath)
		err = Copy(filepath.Join(objectsPath, base+".new"), filePath)
		if err != nil {
			return err
		}

		err = os.Rename(filepath.Join(objectsPath, base+".new"),
			path.Join(objectsPath, base))
		return err
	})

	err = Copy(filepath.Join(repoPath, "json.new"), filepath.Join(p.Pvrdir, "json"))
	if err != nil {
		return err
	}

	return os.Rename(filepath.Join(repoPath, "json.new"),
		path.Join(repoPath, "json"))
}

type Object struct {
	Id         string `json:"id" bson:"id"`
	StorageId  string `json:"storage-id" bson:"_id"`
	Owner      string `json:"owner"`
	ObjectName string `json:"objectname"`
	Sha        string `json:"sha256sum"`
	Size       string `json:"size"`
	MimeType   string `json:"mime-type"`
}

type ObjectWithAccess struct {
	Object       `bson:",inline"`
	SignedPutUrl string `json:"signed-puturl"`
	SignedGetUrl string `json:"signed-geturl"`
	Now          string `json:"now"`
	ExpireTime   string `json:"expire-time"`
}

func (p *Pvr) initializeRemote(repoUrl *url.URL) (pvrapi.PvrRemote, error) {
	res := pvrapi.PvrRemote{}

	pvrRemoteUrl := repoUrl
	pvrRemoteUrl.Path = path.Join(pvrRemoteUrl.Path, ".pvrremote")

	response, err := p.Session.DoAuthCall(true, func(req *resty.Request) (*resty.Response, error) {
		return req.Get(pvrRemoteUrl.String())
	})

	if err != nil {
		return res, err
	}

	if response.StatusCode() != 200 {
		return res, errors.New("REST call failed. " +
			strconv.Itoa(response.StatusCode()) + "  " + response.Status())
	}

	err = json.Unmarshal(response.Body(), &res)

	if err != nil {
		return res, err
	}

	return res, nil

}

// list all objects reffed by current repo json
func listFilesAndObjectsFromJson(json map[string]interface{}, parts []string) (map[string]string, error) {

	filesAndObjects := map[string]string{}
	// push all objects
	for k, v := range json {
		if strings.HasSuffix(k, ".json") {
			continue
		}
		if strings.HasPrefix(k, "#spec") {
			continue
		}
		objId, ok := v.(string)

		found := true
		for _, e := range parts {
			if k == e {
				found = true
				break
			}
			if !strings.HasSuffix(e, "/") {
				e = e + "/"
			}
			if strings.HasPrefix(k, e) {
				found = true
				break
			} else {
				found = false
			}
		}

		if !found {
			continue
		}

		if !ok {
			return map[string]string{}, errors.New("bad object id for file '" + k + "' in pristine pvr json")
		}
		filesAndObjects[k] = objId
	}
	return filesAndObjects, nil
}

// list all objects reffed by current repo json
func (p *Pvr) listFilesAndObjects(parts []string) (map[string]string, error) {

	return listFilesAndObjectsFromJson(p.PristineJsonMap, parts)
}

func readChallenge(targetPrompt string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "*** Claim with challenge @ "+targetPrompt+" ***")
	fmt.Fprint(os.Stderr, "Enter Challenge: ")
	challenge, _ := reader.ReadString('\n')

	return strings.TrimRight(challenge, "\n")
}

func readCredentials(targetPrompt string) (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "*** Login (/type [R] to register) @ "+targetPrompt+" ***")
	fmt.Fprint(os.Stderr, "Username: ")
	username, _ := reader.ReadString('\n')

	username = strings.TrimSpace(username)

	if username == "REGISTER" || username == "REG" || username == "R" {
		return "REGISTER", ""
	}

	fmt.Fprint(os.Stderr, "Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr, "*****")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: "+err.Error())
	}
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

func readRegistration(targetPrompt string) (string, string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "\n*** REGISTER ACCOUNT @ "+targetPrompt+"***")
	fmt.Fprint(os.Stderr, " 1. Email: ")
	email, _ := reader.ReadString('\n')
	fmt.Fprint(os.Stderr, " 2. Username: ")
	username, _ := reader.ReadString('\n')

	password := ""

	for {
		fmt.Fprint(os.Stderr, " 3. Password: ")
		bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr, "*****")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: "+err.Error())
			continue
		}
		password = string(bytePassword)

		fmt.Fprint(os.Stderr, " 4. Password Repeat: ")
		bytePassword, err = terminal.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr, "*****")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: "+err.Error())
			continue
		}
		if password != string(bytePassword) {
			fmt.Fprintln(os.Stderr, "Passwords do not match. Try again!")
			continue
		}
		password = string(bytePassword)
		break
	}

	return strings.TrimSpace(email), strings.TrimSpace(username), strings.TrimSpace(password)
}

func getWwwAuthenticateInfo(header string) (string, map[string]string) {
	parts := strings.SplitN(header, " ", 2)
	authType := strings.TrimSpace(parts[0])
	parts = strings.Split(parts[1], ", ")
	opts := make(map[string]string)

	for _, part := range parts {
		vals := strings.SplitN(part, "=", 2)
		key := strings.ToLower(strings.TrimSpace(vals[0]))
		val := strings.TrimSpace(strings.Trim(vals[1], "\","))
		opts[key] = val
	}
	return authType, opts
}

func (p *Pvr) DoClaim(deviceEp, challenge string) error {

	u, err := url.Parse(deviceEp)

	if err != nil {
		return errors.New("Parsing device URL failed with err=" + err.Error())
	}

	if challenge == "" {
		challenge = readChallenge(deviceEp)
	}

	uV := u.Query()
	uV.Set("challenge", challenge)
	u.RawQuery = uV.Encode()

	response, err := p.Session.DoAuthCall(false, func(req *resty.Request) (*resty.Response, error) {
		return req.Put(u.String())
	})

	if err != nil {
		return errors.New("Claiming device failed with err=" + err.Error())
	}

	if response.StatusCode() != http.StatusOK {
		return errors.New("Claiming device failed (code=" + string(response.StatusCode()) + "): " + string(response.Body()))
	}

	return nil
}

type FilePut struct {
	sourceFile string
	putUrl     string
	objName    string
	objType    string
	res        *http.Response
	err        error
	bar        *pb.ProgressBar
}

const (
	PoolSize = 5
)

type AsyncBody struct {
	Delegate io.ReadCloser
	bar      *pb.ProgressBar
}

func (a *AsyncBody) Read(p []byte) (n int, err error) {

	n, err = a.Delegate.Read(p)
	a.bar.Add(n)

	return n, err
}

func worker(jobs chan FilePut, done chan FilePut) {

	for j := range jobs {
		fstat, err := os.Stat(j.sourceFile)
		if err != nil {
			log.Println("ERROR: " + err.Error())
			continue
		}
		reader, err := os.Open(j.sourceFile)
		if err != nil {
			log.Println("ERROR: " + err.Error())
			continue
		}

		j.bar.Total = fstat.Size()
		j.bar.Units = pb.U_BYTES
		j.bar.UnitsWidth = 25
		j.bar.ShowSpeed = true
		j.bar.ShowCounters = false

		objBaseName := filepath.Base(j.objName)
		j.bar.Prefix(objBaseName[:Min(len(objBaseName)-1, 12)] + " ")

		defer reader.Close()
		r := &AsyncBody{
			Delegate: reader,
			bar:      j.bar,
		}

		req, err := http.NewRequest(http.MethodPut, j.putUrl, r)
		req.ContentLength = fstat.Size()

		if err != nil {
			j.bar.Finish()
			j.err = err
			j.res = nil

			done <- j
			continue
		}

		httpClient := http.DefaultClient

		res, err := httpClient.Do(req)
		j.bar.ShowFinalTime = true
		j.bar.ShowPercent = false
		j.bar.ShowCounters = false
		j.bar.ShowTimeLeft = false
		j.bar.ShowSpeed = false
		j.bar.ShowBar = true
		j.bar.Set64(j.bar.Total)
		j.bar.UnitsWidth = 25
		if err != nil {
			j.bar.Postfix(" [ERROR]")
		} else {
			if j.objType == objects.ObjectTypeLink {
				j.bar.Postfix(" [LK]")
			} else if j.objType == objects.ObjectTypeObject {
				j.bar.Postfix(" [OK]")
			} else {
				j.bar.Postfix(" [OK]")
			}
		}

		j.bar.Finish()

		j.err = err
		j.res = res
		done <- j
	}
}

func (p *Pvr) putFiles(filePut ...FilePut) []FilePut {
	jobs := make(chan FilePut, 100)
	results := make(chan FilePut, 100)

	fileOutPut := []FilePut{}

	pool, err := pb.StartPool()

	if err != nil {
		log.Fatalf("FATAL: starting progressbar pool failed: %s\n", err.Error())
	}

	for i := 0; i < PoolSize; i++ {
		go worker(jobs, results)
	}

	for _, p := range filePut {
		p.bar = pb.New(1)
		pool.Add(p.bar)
		jobs <- p
	}
	close(jobs)

	for i := 0; i < len(filePut); i++ {
		p := <-results
		fileOutPut = append(fileOutPut, p)
		p.bar.Finish()
	}
	close(results)
	pool.Stop()

	return fileOutPut
}

func (p *Pvr) postObjects(pvrRemote pvrapi.PvrRemote, force bool) error {

	var baselineState map[string]interface{}
	var filePutResults []FilePut
	var shaSeen map[string]interface{}
	var filePuts []FilePut
	var filesAndObjects, baselineFilesAndObjects map[string]string
	var refObjects map[string]interface{}
	var err error

	// we have no getUrl in device create and pubobjects case
	if pvrRemote.JsonGetUrl != "" {
		fmt.Fprintf(os.Stderr, "Synching baseline state with device")

		buf, err := p.getJSONBuf(pvrRemote)

		if err != nil {
			goto errout
		}

		json.Unmarshal(buf, &baselineState)

		if err != nil {
			goto errout
		}
		fmt.Fprintf(os.Stderr, " [OK]\n")
	}

	baselineFilesAndObjects, err = listFilesAndObjectsFromJson(baselineState, []string{})

	if err != nil {
		return err
	}

	refObjects = map[string]interface{}{}
	for _, v := range baselineFilesAndObjects {
		refObjects[v] = true
	}

	filesAndObjects, err = p.listFilesAndObjects([]string{})
	if err != nil {
		return err
	}

	filePuts = []FilePut{}

	shaSeen = map[string]interface{}{}

	// push all objects
	for k, v := range filesAndObjects {

		// we skip objects already in baseline state
		if refObjects[v] != nil {
			continue
		}

		info, err := os.Stat(filepath.Join(p.Objdir, v))
		if err != nil {
			return err
		}
		sizeString := fmt.Sprintf("%d", info.Size())

		remoteObject := ObjectWithAccess{}
		remoteObject.Object.Size = sizeString
		remoteObject.MimeType = "application/octet-stream"
		remoteObject.Sha = v
		remoteObject.ObjectName = k

		uri := pvrRemote.ObjectsEndpointUrl
		if !strings.HasSuffix(uri, "/") {
			uri += "/"
		}

		fmt.Fprintf(os.Stderr, "Posting object info for: "+k+" ... ")

		response, err := p.Session.DoAuthCall(false, func(req *resty.Request) (*resty.Response, error) {
			res, err := req.SetBody(remoteObject).Post(uri)

			if err != nil {
				fmt.Fprintf(os.Stderr, " [ERROR: "+err.Error()+"]")
			}
			return res, err
		})

		if err != nil {
			goto errout
		}

		if response == nil {
			err = errors.New("BAD STATE; no respo")
			goto errout
		}

		if shaSeen[remoteObject.Sha] != nil {
			_str := remoteObject.ObjectName[0:Min(len(remoteObject.ObjectName)-1, 12)] + " "
			fmt.Fprintln(os.Stderr, _str+"[OK - Dupe]")
			continue
		}
		shaSeen[remoteObject.Sha] = "yes"

		if response.StatusCode() != http.StatusOK &&
			response.StatusCode() != http.StatusConflict {
			err = errors.New("Error posting object " + strconv.Itoa(response.StatusCode()))
			goto errout
		}

		if response.StatusCode() == http.StatusConflict {
			_str := remoteObject.ObjectName[0:Min(len(remoteObject.ObjectName)-1, 12)] + " "
			objectType := response.Header().Get(objects.HttpHeaderPantahubObjectType)
			if !force {
				if objectType == objects.ObjectTypeLink {
					fmt.Fprintln(os.Stderr, _str+"[LK]")
				} else if objectType == objects.ObjectTypeObject {
					fmt.Fprintln(os.Stderr, _str+"[OK]")
				} else {
					fmt.Fprintln(os.Stderr, _str+"[OK]")
				}
				continue
			}

			// if force
			if objectType == objects.ObjectTypeLink {
				fmt.Fprintln(os.Stderr, _str+"[LK]")
				continue
			}
		}

		err = json.Unmarshal(response.Body(), &remoteObject)
		if err != nil {
			goto errout
		}

		fileName := filepath.Join(p.Objdir, v)
		filePut := FilePut{
			sourceFile: fileName,
			objName:    remoteObject.ObjectName,
			putUrl:     remoteObject.SignedPutUrl,
		}

		if response.StatusCode() == http.StatusConflict && force {
			objectType := response.Header().Get(objects.HttpHeaderPantahubObjectType)

			// code in 'force' case will only get here if its not an ObjectTypeLink
			// object type links are filtered and acked further above in this loop.
			filePut.objType = objectType
		} else {
			filePut.objType = objects.ObjectTypeObject
		}
		filePuts = append(filePuts, filePut)
		fmt.Fprintln(os.Stderr, " [OK]")
	}

	filePutResults = p.putFiles(filePuts...)

	for _, v := range filePutResults {
		if v.err != nil {
			err = fmt.Errorf("Error putting file %s: %s", v.objName, v.err.Error())
			goto errout
		}
		if 200 != v.res.StatusCode {
			err = errors.New("REST call failed. " +
				strconv.Itoa(v.res.StatusCode) + "  " + v.res.Status)
			goto errout
		}
	}

	return nil

errout:
	fmt.Fprintf(os.Stderr, " [ERROR: "+err.Error()+"]")
	return err
}

func (p *Pvr) PutRemote(repoPath *url.URL, force bool) error {

	pvrRemote, err := p.initializeRemote(repoPath)

	if err != nil {
		return err
	}

	err = p.postObjects(pvrRemote, force)

	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(filepath.Join(p.Pvrdir, "json"))

	if err != nil {
		return err
	}

	uri := pvrRemote.JsonGetUrl
	body := map[string]interface{}{}
	err = json.Unmarshal(data, &body)

	if err != nil {
		return err
	}

	response, err := p.Session.DoAuthCall(false, func(req *resty.Request) (*resty.Response, error) {
		return req.SetBody(body).Put(uri)
	})

	if err != nil {
		return err
	}

	if 200 != response.StatusCode() {
		return errors.New("REST call failed. " +
			strconv.Itoa(response.StatusCode()) + "  " + response.Status() + "\n\n   " + string(response.Body()))
	}

	err = json.Unmarshal(response.Body(), &body)

	if err != nil {
		return err
	}

	return nil
}

func (p *Pvr) Put(uri string, force bool) error {

	if uri == "" {
		uri = p.Pvrconfig.DefaultPutUrl
	}

	url, err := url.Parse(uri)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		_, err := os.Stat(filepath.Join(uri, "json"))
		// if we get pointed at a pvr repo on disk, go local
		if err == nil {
			err = p.PutLocal(uri)
			goto save
		} else if !os.IsNotExist(err) {
			return errors.New("error testing existance of json file in provided path: " + err.Error())
		}
		repoBaseURL, err := url.Parse(p.Session.GetApp().Metadata["PVR_REPO_BASEURL"].(string))
		if err != nil {
			return errors.New("error parsing PVR_REPO_BASEURL setting, see --help - ERROR:" + err.Error())
		}

		if !path.IsAbs(uri) {
			uri = "/" + uri
		}

		refURL, err := url.Parse(uri)
		if err != nil {
			return errors.New("error parsing provided repo name, see --help - ERROR:" + err.Error())
		}

		url = repoBaseURL.ResolveReference(refURL)

	}

	err = p.PutRemote(url, force)

	if err != nil {
		return err
	}

save:
	if p.Pvrconfig.DefaultGetUrl == "" {
		p.Pvrconfig.DefaultGetUrl = uri
	}
	if p.Pvrconfig.DefaultPutUrl == "" {
		p.Pvrconfig.DefaultPostUrl = uri
	}
	if err == nil {
		p.Pvrconfig.DefaultPutUrl = uri
		err = p.SaveConfig()
	}

	return err
}

func (p *Pvr) SaveConfig() error {
	configNew := filepath.Join(p.Pvrdir, "config.new")
	configPath := filepath.Join(p.Pvrdir, "config")

	byteJson, err := json.MarshalIndent(p.Pvrconfig, "", "	")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(configNew, byteJson, 0644)
	if err != nil {
		return err
	}
	err = os.Rename(configNew, configPath)
	return err
}

func (p *Pvr) PutObjects(uri string, force bool) error {
	url, err := url.Parse(uri)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		return errors.New("not implemented PutObjects Local")
	}

	pvr := pvrapi.PvrRemote{
		ObjectsEndpointUrl: uri,
	}

	return p.postObjects(pvr, force)
}

func (p *Pvr) postRemoteJson(remotePvr pvrapi.PvrRemote, pvrMap PvrMap, envelope string,
	commitMsg string, rev int, force bool) ([]byte, error) {

	if envelope == "" {
		envelope = "{}"
	}

	envJSON := map[string]interface{}{}
	err := json.Unmarshal([]byte(envelope), &envJSON)

	if err != nil {
		return nil, err
	}

	if commitMsg != "" {
		envJSON["commit-msg"] = commitMsg
	}

	if rev != 0 {
		envJSON["rev"] = rev
	}

	if remotePvr.JsonKey != "" {
		envJSON[remotePvr.JsonKey] = pvrMap
	} else {
		envJSON["post"] = pvrMap
	}

	data, err := cjson.Marshal(envJSON)

	if err != nil {
		return nil, err
	}

	response, err := p.Session.DoAuthCall(false, func(req *resty.Request) (*resty.Response, error) {
		return req.SetBody(data).SetContentLength(true).Post(remotePvr.PostUrl)
	})

	if err != nil {
		return nil, err
	}

	if response.StatusCode() != 200 {
		return nil, errors.New("REST call failed. " +
			strconv.Itoa(response.StatusCode()) + "  " + response.Status() +
			"\n\t" + string(response.Body()))
	}

	return response.Body(), nil
}

// make a json post to a REST endpoint. You can provide metainfo etc. in post
// argument as json. postKey if set will be used as key that refers to the posted
// json. Example usage: json blog post, json revision repo with commit message etc
func (p *Pvr) Post(uri string, envelope string, commitMsg string, rev int, force bool) error {

	if uri == "" {
		uri = p.Pvrconfig.DefaultPostUrl
	}
	url, err := url.Parse(uri)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		_, err := os.Stat(filepath.Join(uri, "json"))
		// if we get pointed at a pvr repo on disk, go local
		if err == nil {
			return errors.New("Post must be a remote REST endpoint, not: " + uri)
		} else if !os.IsNotExist(err) {
			return errors.New("error testing existance of json file in provided path: " + err.Error())
		}

		repoBaseURL, err := url.Parse(p.Session.GetApp().Metadata["PVR_REPO_BASEURL"].(string))
		if err != nil {
			return errors.New("error parsing PVR_REPO_BASEURL setting, see --help - ERROR:" + err.Error())
		}

		if !path.IsAbs(uri) {
			uri = "/" + uri
		}

		refURL, err := url.Parse(uri)
		if err != nil {
			return errors.New("error parsing provided repo name, see --help - ERROR:" + err.Error())
		}

		url = repoBaseURL.ResolveReference(refURL)
	}

	remotePvr, err := p.initializeRemote(url)

	if err != nil {
		return err
	}

	err = p.postObjects(remotePvr, force)

	if err != nil {
		return err
	}

	body, err := p.postRemoteJson(remotePvr, p.PristineJsonMap, envelope, commitMsg, rev, force)

	if err != nil {
		return err
	}

	responseMap := map[string]interface{}{}
	err = json.Unmarshal(body, &responseMap)
	if err != nil {
		return err
	}

	stateSha := responseMap["state-sha"].(string)
	revLocal := fmt.Sprintf("%d", int(responseMap["rev"].(float64)))
	if revLocal == "-1" {
		revLocal = responseMap["revlocal"].(string)
	}

	fmt.Fprintf(os.Stderr, "Successfully posted Revision %s (%s) to device id %s\n", revLocal,
		stateSha[:Min(8, len(stateSha))], responseMap["trail-id"])

	p.Pvrconfig.DefaultPostUrl = uri
	if p.Pvrconfig.DefaultGetUrl == "" {
		p.Pvrconfig.DefaultGetUrl = uri
	}

	if p.Pvrconfig.DefaultPutUrl == "" {
		p.Pvrconfig.DefaultPutUrl = uri
	}

	err = p.SaveConfig()

	if err != nil {
		fmt.Fprintln(os.Stderr, "WARNING: couldnt save config "+err.Error())
	}

	return nil
}

func (p *Pvr) UnpackRepo(repoPath string, outDir string) error {

	err := Untar(outDir, repoPath)
	return err
}

func (p *Pvr) GetStateJson(uri string) (
	state map[string]interface{},
	err error,
) {
	var u *url.URL
	var data []byte

	if uri == "" {
		uri = p.Pvrconfig.DefaultGetUrl
	}

	u, err = url.Parse(uri)

	if err != nil {
		return
	}

	exists, err := IsFileExists(uri)

	if err != nil {
		return
	}

	if u.Scheme == "" && exists {
		var fileInfo os.FileInfo
		repoPath := uri

		fileInfo, err = os.Stat(repoPath)
		if err != nil {
			return
		}

		// if we dont have a dir for local we might have a tarball export
		if !fileInfo.IsDir() {
			repoPath, err = ioutil.TempDir(os.TempDir(), "pvr-tmprepo-")
			if err != nil {
				return
			}
			defer os.RemoveAll(repoPath)

			err = p.UnpackRepo(uri, repoPath)
			if err != nil {
				return
			}
		}

		localJsonPath := filepath.Join(repoPath, "json")
		data, err = ioutil.ReadFile(localJsonPath)
		if err != nil {
			return
		}
		err = json.Unmarshal(data, &state)
		if err != nil {
			return
		}
	} else {

		if u.Scheme == "" {
			u, err = url.Parse("https://pvr.pantahub.com/" + uri)
		}

		if err != nil {
			return
		}

		var remote pvrapi.PvrRemote
		remote, err = p.initializeRemote(u)
		if err != nil {
			return
		}
		data, err = p.getJSONBuf(remote)
		if err != nil {
			return
		}
		err = json.Unmarshal(data, &state)
		if err != nil {
			return
		}
	}

	return
}

func (p *Pvr) GetRepoLocal(getPath string, merge bool, showFilenames bool) (
	objectsCount int,
	err error) {
	jsonMap := map[string]interface{}{}

	objectsCount = 0

	repoUri, err := url.Parse(getPath)

	if err != nil {
		return objectsCount, err
	}

	repoPath := repoUri.Path

	// lets keep only those matching prefix
	partPrefixes := []string{}
	unpartPrefixes := []string{}
	if repoUri.Fragment != "" {
		parsePrefixes := strings.Split(repoUri.Fragment, ",")
		for _, v := range parsePrefixes {
			if !strings.HasPrefix(v, "-") {
				partPrefixes = append(partPrefixes, v)
			} else {
				unpartPrefixes = append(unpartPrefixes, v[1:])
			}
		}
	}

	f, err := os.Stat(repoPath)

	if err != nil {
		return objectsCount, err
	}

	// if we dont have a dir for local we might have a tarball export
	if !f.IsDir() {
		repoPath, err = ioutil.TempDir(os.TempDir(), "pvr-tmprepo-")
		if err != nil {
			return objectsCount, err
		}
		defer os.RemoveAll(repoPath)

		err = p.UnpackRepo(repoUri.Path, repoPath)
		if err != nil {
			return objectsCount, err
		}
	}

	// first copy new json, but only rename at the very end after all else succeed
	jsonRepo := filepath.Join(repoPath, "json")

	_, err = os.Stat(jsonRepo)
	if err != nil {
		return objectsCount, err
	}

	jsonData, err := ioutil.ReadFile(jsonRepo)
	if err != nil {
		return objectsCount, err
	}

	err = json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		return objectsCount, errors.New("JSON Unmarshal (json.new):" + err.Error())
	}

	// delete keys that have no prefix
	for k := range jsonMap {
		found := true
		for _, v := range partPrefixes {
			if k == v {
				found = true
				break
			}
			if !strings.HasSuffix(v, "/") {
				v += "/"
			}
			if strings.HasPrefix(k, v) {
				found = true
				break
			} else {
				found = false
			}
		}
		if !found {
			delete(jsonMap, k)
		}
	}

	// first copy new json, but only rename at the very end after all else succeed
	configRepo := filepath.Join(repoPath, "config")

	var config *PvrConfig

	configData, _ := ioutil.ReadFile(configRepo)
	config = new(PvrConfig)
	// keep default config if there is no config file yet
	if configData != nil {
		err = json.Unmarshal(configData, config)
		if err != nil {
			return objectsCount, errors.New("JSON Unmarshal (config.new):" + err.Error())
		}
	}

	for k, v := range jsonMap {
		if strings.HasSuffix(k, ".json") {
			continue
		}
		if strings.HasPrefix(k, "#spec") {
			continue
		}
		getPath := filepath.Join(repoPath, "objects", v.(string))

		// config can overload objectsdir for remote repo; not abs and abs
		if config != nil && config.ObjectsDir != "" {
			if !filepath.IsAbs(config.ObjectsDir) {
				getPath = filepath.Join(repoPath, "..", config.ObjectsDir, v.(string))
			} else {
				getPath = filepath.Join(config.ObjectsDir, v.(string))
			}
		}

		objPathNew := filepath.Join(p.Objdir, v.(string)+".new")
		objPath := filepath.Join(p.Objdir, v.(string))
		fileExists, err := IsFileExists(objPath)

		if err != nil {
			return objectsCount, err
		}

		if showFilenames {
			cache := " cache"
			if !fileExists {
				cache = ""
			}

			if len(k) >= 15 {
				fmt.Fprintln(os.Stderr, k[:15]+" [OK"+cache+"]")
			} else {
				fmt.Fprintln(os.Stderr, k+" [OK"+cache+"]")
			}

		} else {
			fmt.Fprintln(os.Stderr, "pulling objects file "+getPath+"-> "+objPathNew)
		}

		err = Copy(objPathNew, getPath)

		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR : "+err.Error())
			return objectsCount, err
		}
		err = os.Rename(objPathNew, objPath)
		if err != nil {
			return objectsCount, err
		}
		objectsCount++
	}

	var jsonMerged []byte
	if merge {
		jsonDataSelect, err := cjson.Marshal(jsonMap)

		if err != nil {
			return objectsCount, err
		}
		jsonMerged, err = jsonpatch.MergePatchIndent(p.PristineJson, jsonDataSelect, "", "	")
	} else {
		// manually remove everything not matching the part from fragement ...
		pJSONMap := p.PristineJsonMap

		if len(partPrefixes) == 0 {
			partPrefixes = append(partPrefixes, "")
		}

		for _, partPrefix := range partPrefixes {
			// remove all files for name "app/"
			for k := range pJSONMap {
				if strings.HasPrefix(k, partPrefix) {
					delete(pJSONMap, k)
				}
			}
			// add back all from new map
			for k, v := range jsonMap {
				if strings.HasPrefix(k, partPrefix) {
					pJSONMap[k] = v
				}
			}
		}
		for _, unpartPrefix := range unpartPrefixes {
			// remove all files for name "app/"
			for k := range pJSONMap {
				if strings.HasPrefix(k, unpartPrefix) {
					delete(pJSONMap, k)
				}
			}
		}

		jsonMerged, err = cjson.Marshal(pJSONMap)
	}

	if err != nil {
		return objectsCount, err
	}

	p.PristineJson = jsonMerged
	err = json.Unmarshal(p.PristineJson, &p.PristineJsonMap)

	if err != nil {
		return objectsCount, err
	}

	jsonMerged, err = cjson.Marshal(p.PristineJsonMap)

	if err != nil {
		return objectsCount, err
	}

	err = ioutil.WriteFile(filepath.Join(p.Pvrdir, "json.new"), jsonMerged, 0644)

	if err != nil {
		return objectsCount, err
	}

	// all succeeded, atomically commiting the json
	err = os.Rename(filepath.Join(p.Pvrdir, "json.new"), filepath.Join(p.Pvrdir, "json"))

	return objectsCount, err
}

func (p *Pvr) getJSONBuf(pvrRemote pvrapi.PvrRemote) ([]byte, error) {

	response, err := p.Session.DoAuthCall(true, func(req *resty.Request) (*resty.Response, error) {
		return req.Get(pvrRemote.JsonGetUrl)
	})

	if err != nil {
		return nil, err
	}

	jsonData := response.Body()

	return jsonData, nil
}

func (p *Pvr) grabObjects(showFilenames bool, requests ...*grab.Request) (
	objectsCount int,
	err error,
) {
	client := grab.NewClient()
	client.HTTPClient.Transport = http.DefaultTransport

	client.UserAgent = "PVR client"
	respch := client.DoBatch(4, requests...)

	// start a ticker to update progress every 200ms
	t := time.NewTicker(200 * time.Millisecond)

	progressBars := map[*grab.Request]*pb.ProgressBar{}
	progressBarSlice := make([]*pb.ProgressBar, 0)

	for _, v := range requests {
		if showFilenames {
			fileName := v.Label
			if len(v.Label) >= 15 {
				fileName = v.Label[:15]
			}
			progressBars[v] = pb.New(0).Prefix(fileName)
		} else {
			shortFile := filepath.Base(v.Filename)[:15]
			progressBars[v] = pb.New(0).Prefix(shortFile)
		}
		progressBars[v].ShowCounters = false
		progressBars[v].SetUnits(pb.KB)
		progressBarSlice = append(progressBarSlice, progressBars[v])
	}

	// monitor downloads
	completed := 0
	inProgress := 0
	responses := make([]*grab.Response, 0)

	pool, err := pb.StartPool(progressBarSlice...)
	if err != nil {
		goto err
	}

	for completed < len(requests) {
		select {
		case resp := <-respch:
			// a new response has been received and has started downloading
			// (nil is received once, when the channel is closed by grab)
			if resp != nil {
				responses = append(responses, resp)
			}

		case <-t.C:
			// update completed downloads
			for i, resp := range responses {
				if resp != nil && resp.IsComplete() {
					req := resp.Request
					// print final result
					if resp.Err() != nil && grab.ErrFileExists != resp.Err() {
						log.Println("ERROR: Downloading " + resp.Err().Error())
						progressBars[req].Finish()
						progressBars[req].ShowBar = false
						progressBars[req].ShowPercent = false
						progressBars[req].ShowCounters = false
						progressBars[req].ShowFinalTime = false
						goto err

					} else {
						progressBars[req].Finish()
						progressBars[req].ShowFinalTime = false
						progressBars[req].ShowPercent = false
						progressBars[req].ShowCounters = false
						progressBars[req].ShowTimeLeft = false
						progressBars[req].ShowBar = false
						if grab.ErrFileExists == resp.Err() {
							if req.Tag == objects.ObjectTypeLink {
								progressBars[req].Postfix(" [LK cache]")
							} else if req.Tag == objects.ObjectTypeObject {
								progressBars[req].Postfix(" [OK cache]")
							} else {
								progressBars[req].Postfix(" [OK cache]")
							}
						} else if req.Tag == objects.ObjectTypeLink {
							progressBars[req].Postfix(" [LK]")
						} else if req.Tag == objects.ObjectTypeObject {
							progressBars[req].Postfix(" [OK]")
						} else {
							progressBars[req].Postfix(" [OK]")
						}
						progressBars[req].Set64(progressBars[req].Total)
					}

					// mark completed
					responses[i] = nil
					completed++
				}
			}

			// update downloads in progress
			inProgress = 0
			for _, resp := range responses {
				if resp != nil {
					req := resp.Request
					inProgress++
					progressBars[req].Total = resp.HTTPResponse.ContentLength
					progressBars[req].ShowTimeLeft = true
					progressBars[req].ShowCounters = true
					progressBars[req].ShowPercent = true
					progressBars[req].Set64(resp.BytesComplete())
					progressBars[req].SetUnits(pb.KB)
				}
			}
		}
	}

err:
	if pool != nil {
		pool.Stop()
	}
	t.Stop()

	objectsCount = completed

	return objectsCount, nil
}

func (p *Pvr) getObjects(showFilenames bool, pvrRemote pvrapi.PvrRemote, jsonMap map[string]interface{}) (
	objectsCount int,
	err error,
) {

	objectsCount = 0

	grabs := make([]*grab.Request, 0)
	shaMap := map[string]interface{}{}

	for k := range jsonMap {
		var req *grab.Request
		var remoteObject ObjectWithAccess
		var response *resty.Response
		var uri string

		if strings.HasSuffix(k, ".json") {
			continue
		}
		if strings.HasPrefix(k, "#spec") {
			continue
		}

		v := jsonMap[k].(string)

		fullPathV := path.Join(p.Objdir, v)

		fSha, err := FiletoSha(fullPathV)
		if err != nil && !os.IsNotExist(err) {
			log.Println("ERROR: error calculating sha for existing file " + fullPathV + ": " + err.Error())
			return objectsCount, err
		}

		if err == nil && fSha != v {
			err = os.Remove(fullPathV)
			if err != nil {
				log.Println("WARNING: error removing not sha-matching local object: " + fullPathV + " - " + err.Error())
			}
		}

		if err == nil && fSha == v {
			// skip objects for which we have a valid sha in pool
			continue
		}

		fmt.Fprintf(os.Stderr, "Getting object info for: "+k+" ... ")

		// only add to downloads if we have not seen this sha already
		if shaMap[v] != nil {
			goto cont
		} else {
			shaMap[v] = "seen"
		}

		uri = pvrRemote.ObjectsEndpointUrl + "/" + v

		response, err = p.Session.DoAuthCall(true, func(req *resty.Request) (*resty.Response, error) {
			return req.Get(uri)
		})

		if err != nil {
			return objectsCount, err
		}

		if response.StatusCode() != 200 {
			return objectsCount, errors.New("REST call failed. " +
				strconv.Itoa(response.StatusCode()) + "  " + response.Status())
		}

		err = json.Unmarshal(response.Body(), &remoteObject)

		if err != nil {
			return objectsCount, err
		}

		// we grab them to .new file ... and rename them when completed
		req, err = grab.NewRequest(fullPathV, remoteObject.SignedGetUrl)
		if err != nil {
			return objectsCount, err
		}
		req.SkipExisting = true
		req.Tag = response.Header().Get(objects.HttpHeaderPantahubObjectType)
		req.Label = remoteObject.ObjectName

		grabs = append(grabs, req)

	cont:
		fmt.Fprintf(os.Stderr, "OK\n")
	}

	objectsCount, err = p.grabObjects(showFilenames, grabs...)

	return objectsCount, err
}

func (p *Pvr) GetRepoRemote(url *url.URL, merge bool, showFilenames bool) (
	objectsCount int,
	err error,
) {

	objectsCount = 0

	if url.Scheme == "" {
		return objectsCount, errors.New("Post must be a remote REST endpoint, not: " + url.String())
	}

	remotePvr, err := p.initializeRemote(url)
	if err != nil {
		return objectsCount, err
	}

	jsonData, err := p.getJSONBuf(remotePvr)

	if err != nil {
		return objectsCount, err
	}

	jsonMap := map[string]interface{}{}

	err = json.Unmarshal(jsonData, &jsonMap)
	if err != nil {
		return objectsCount, err
	}

	// lets keep only those matching prefix
	partPrefixes := []string{}
	unpartPrefixes := []string{}
	if url.Fragment != "" {
		parsePrefixes := strings.Split(url.Fragment, ",")
		for _, v := range parsePrefixes {
			if !strings.HasPrefix(v, "-") {
				partPrefixes = append(partPrefixes, v)
			} else {
				unpartPrefixes = append(unpartPrefixes, v[1:])
			}
		}
	}

	// delete keys that have no prefix
	for k := range jsonMap {
		found := true
		for _, v := range partPrefixes {
			if k == v {
				found = true
				break
			}
			if !strings.HasSuffix(v, "/") {
				v += "/"
			}
			if strings.HasPrefix(k, v) {
				found = true
				break
			} else {
				found = false
			}
		}
		if !found {
			delete(jsonMap, k)
		}
	}

	objectsCount, err = p.getObjects(showFilenames, remotePvr, jsonMap)

	if err != nil {
		return objectsCount, err
	}

	var jsonMerged []byte
	if merge {
		jsonDataSelect, err := cjson.Marshal(jsonMap)
		if err != nil {
			return objectsCount, err
		}

		pJSONMap := p.PristineJsonMap

		for _, unpartPrefix := range unpartPrefixes {
			// remove all files for name "app/"
			for k := range pJSONMap {
				if strings.HasPrefix(k, unpartPrefix) {
					delete(pJSONMap, k)
				}
			}
		}

		jsonMerged, err = jsonpatch.MergePatch(p.PristineJson, jsonDataSelect)

	} else {
		// manually remove everything not matching the part from fragment ...
		pJSONMap := p.PristineJsonMap

		if len(partPrefixes) == 0 {
			partPrefixes = append(partPrefixes, "")
		}

		for _, partPrefix := range partPrefixes {
			// remove all files for name "app/"
			for k := range pJSONMap {
				if strings.HasPrefix(k, partPrefix) {
					delete(pJSONMap, k)
				}
			}
			// add back all from new map
			for k, v := range jsonMap {
				if strings.HasPrefix(k, partPrefix) {
					pJSONMap[k] = v
				}
			}
		}

		for _, unpartPrefix := range unpartPrefixes {
			// remove all files for name "app/"
			for k := range pJSONMap {
				if strings.HasPrefix(k, unpartPrefix) {
					delete(pJSONMap, k)
				}
			}
		}

		jsonMerged, err = cjson.Marshal(pJSONMap)
	}

	if err != nil {
		return objectsCount, err
	}

	p.PristineJson = jsonMerged
	err = json.Unmarshal(p.PristineJson, &p.PristineJsonMap)

	if err != nil {
		return objectsCount, err
	}

	jsonMerged, err = cjson.Marshal(p.PristineJsonMap)

	if err != nil {
		return objectsCount, err
	}

	err = ioutil.WriteFile(filepath.Join(p.Pvrdir, "json.new"), jsonMerged, 0644)

	if err != nil {
		return objectsCount, err
	}

	return objectsCount, os.Rename(filepath.Join(p.Pvrdir, "json.new"), filepath.Join(p.Pvrdir, "json"))
}

func (p *Pvr) GetRepo(uri string, merge bool, showFilenames bool) (
	objectsCount int,
	err error,
) {
	objectsCount = 0

	if uri == "" {
		uri = p.Pvrconfig.DefaultPutUrl
	}

	url, err := url.Parse(uri)

	if err != nil {
		return objectsCount, err
	}

	p.Pvrconfig.DefaultGetUrl = uri

	// if no url scheme try following in order;
	//  1. is uri a local .pvr repo directory -> GetRepoLocal
	//  2. if a path with one or two elements -> Prepend https://pvr.pantahub.com
	//  3. if a first dir of path is resolvable host -> Prepend https://
	if url.Scheme == "" {
		objectsCount, err = p.GetRepoLocal(uri, merge, showFilenames)

		// if we get pointed at a pvr repo on disk, go local
		if err == nil {
			goto save
		} else if !os.IsNotExist(err) {
			return objectsCount, errors.New("error testing existance of local json file in provided path: " + err.Error())
		}

		repoBaseURL, err := url.Parse(p.Session.GetApp().Metadata["PVR_REPO_BASEURL"].(string))
		if err != nil {
			return objectsCount, errors.New("error parsing PVR_REPO_BASEURL setting, see --help - ERROR:" + err.Error())
		}

		if !path.IsAbs(url.Path) {
			uri = "/" + uri
		}

		url = repoBaseURL.ResolveReference(url)

	}

	if p.Pvrconfig.DefaultPutUrl == "" {
		p.Pvrconfig.DefaultPutUrl = uri
	}

	if p.Pvrconfig.DefaultPostUrl == "" {
		p.Pvrconfig.DefaultPostUrl = uri
	}

	objectsCount, err = p.GetRepoRemote(url, merge, showFilenames)

	if err != nil {
		return objectsCount, err
	}

save:
	err = p.SaveConfig()

	return objectsCount, err
}

func (p *Pvr) Reset(canonicalJson bool) error {
	return p.resetInternal(false, canonicalJson)
}

func (p *Pvr) ResetWithHardlink() error {
	return p.resetInternal(true, true)
}

func (p *Pvr) resetInternal(hardlink bool, canonicalJson bool) error {
	data, err := ioutil.ReadFile(filepath.Join(p.Pvrdir, "json"))

	if err != nil {
		return err
	}
	jsonMap := map[string]interface{}{}

	err = json.Unmarshal(data, &jsonMap)

	if err != nil {
		return errors.New("JSON Unmarshal (" +
			strings.TrimPrefix(filepath.Join(p.Pvrdir, "json"), p.Dir) + "): " +
			err.Error())
	}

	for k, v := range jsonMap {
		if strings.HasPrefix(k, "#spec") {
			continue
		}

		fromSlashK := filepath.FromSlash(k)
		targetP := filepath.Join(p.Dir, fromSlashK)
		targetD := path.Dir(targetP)
		targetDInfo, err := os.Stat(targetD)
		if err != nil {
			err = os.MkdirAll(targetD, 0755)
		} else if !targetDInfo.IsDir() {
			return errors.New("Not a directory " + targetD)
		}
		if err != nil {
			return err
		}

		if strings.HasSuffix(k, ".json") {
			var data []byte
			var err error

			// if ! hardlink then we checkout as developer copy
			// lets make reading this a pleasure; if however
			// we are in hardlink mode then it makes sense to
			// assume that the user wants the checked out file
			// to match exactly what is in the pvr json
			if !hardlink && !canonicalJson {
				data, err = json.MarshalIndent(v, "", "    ")
			} else {
				data, err = cjson.Marshal(v)
			}
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(targetP+".new", data, 0644)
			if err != nil {
				return err
			}
			err = os.Rename(targetP+".new",
				targetP)
		} else {
			objectP := filepath.Join(p.Objdir, v.(string))
			if !hardlink {
				err = Copy(targetP+".new", objectP)
				if err != nil {
					return err
				}
				err = os.Rename(targetP+".new", targetP)
				if err != nil {
					return err
				}
			} else {
				err = Hardlink(targetP, objectP)
			}
		}
	}
	os.Remove(filepath.Join(p.Pvrdir, "new"))
	return nil
}

func addToTar(writer *tar.Writer, archivePath, sourcePath string) error {

	stat, err := os.Stat(sourcePath)

	if err != nil {
		return err
	}

	if stat.IsDir() {
		return errors.New("pvr repo broken state: object file '" + sourcePath + "'is a directory")
	}

	object, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer object.Close()

	header := new(tar.Header)
	header.Name = archivePath
	header.Size = stat.Size()
	header.Mode = int64(stat.Mode())
	header.ModTime = stat.ModTime()

	err = writer.WriteHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, object)
	if err != nil {
		return err
	}

	return nil
}

// Export will put the 'json' file first into the archive to allow for
// stream parsing and validation of json before processing objects
func (p *Pvr) Export(parts []string, dst string) error {

	var file *os.File
	var err error

	if dst == "-" {
		file = os.Stdout
	} else {
		file, err = os.Create(dst)
		if err != nil {
			return err
		}
		defer file.Close()
	}

	var fileWriter io.WriteCloser

	// stdout and
	if strings.HasSuffix(strings.ToLower(dst), ".gz") ||
		strings.HasSuffix(strings.ToLower(dst), ".tgz") {

		fileWriter = gzip.NewWriter(file)
		if err != nil {
			return err
		}
		defer fileWriter.Close()
	} else {
		fileWriter = file
	}

	tw := tar.NewWriter(fileWriter)
	defer tw.Close()

	filteredMap := map[string]interface{}{}

	for k, v := range p.PristineJsonMap {
		found := true
		for _, p := range parts {
			// full key match (explicit file part) is here
			if k == p {
				found = true
				break
			}
			if !strings.HasSuffix(p, "/") {
				// no full match lets ensure we only match full directories
				p = p + "/"
			}
			if strings.HasPrefix(k, p) {
				found = true
				break
			}
			found = false
		}
		if k == "#spec" {
			found = true
		}
		if found {
			filteredMap[k] = v
		}
	}
	jsonFile, err := ioutil.TempFile(os.TempDir(), "filtered-json.json.XXXXXXXX")
	if err != nil {
		return err
	}

	defer jsonFile.Close()
	defer os.Remove(jsonFile.Name())

	buf, err := cjson.Marshal(filteredMap)

	if err != nil {
		return err
	}

	_, err = jsonFile.Write(buf)

	if err != nil {
		return err
	}

	if err := addToTar(tw, "json", jsonFile.Name()); err != nil {
		return err
	}

	filesAndObjects, err := p.listFilesAndObjects(parts)
	if err != nil {
		return err
	}

	for _, v := range filesAndObjects {
		apath := "objects/" + v
		ipath := filepath.Join(p.Objdir, v)
		err := addToTar(tw, apath, ipath)

		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pvr) Import(src string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	var fileReader io.ReadCloser

	if strings.HasSuffix(strings.ToLower(src), ".gz") ||
		strings.HasSuffix(strings.ToLower(src), ".tgz") {

		fileReader, err = gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer fileReader.Close()
	} else {
		fileReader = file
	}

	tw := tar.NewReader(fileReader)

	for {
		header, err := tw.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fileInfo := header.FileInfo()

		// we do not make directories as the only directory
		// .pvr/objects must exist in inititialized pvrs
		if fileInfo.IsDir() {
			continue
		}

		var filePath string
		if filepath.Base(filepath.Dir(header.Name)) == "objects" {
			filePath = filepath.Join(p.Objdir, filepath.Base(header.Name))
		} else {
			filePath = filepath.Join(p.Pvrdir, header.Name)
		}
		filePathNew := filePath + ".new"

		file, err := os.OpenFile(filePathNew, os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
			fileInfo.Mode())
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, tw)
		if err != nil {
			return err
		}
		err = os.Rename(filePathNew, filePath)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeployPvLinks sets up the hardlinks for the .pv/ files
func (p *Pvr) DeployPvLinks() error {

	var buf []byte
	var err error
	jsonMap := map[string]interface{}{}

	bspRunJSON := path.Join(p.Dir, "bsp", "run.json")
	buf, err = ioutil.ReadFile(bspRunJSON)

	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, &jsonMap)
	if err != nil {
		return err
	}

	fitFileI := jsonMap["fit"]
	kernelFileI := jsonMap["kernel"]
	initrdFile := "pantavisor"
	dtbFileI := jsonMap["fdt"]

	var kernelFile string
	var fitFile string
	if fitFileI != nil {
		fitFile = fitFileI.(string)
	} else if kernelFileI != nil {
		kernelFile = kernelFileI.(string)
	} else {
		kernelFile = "kernel.img"
	}

	var dtbFile string
	if dtbFileI != nil {
		dtbFile = dtbFileI.(string)
	}

	if fitFile != "" {
		fitLink := path.Join(p.Dir, ".pv", "pantavisor.fit")
		os.Mkdir(path.Join(p.Dir, ".pv"), 0755)
		os.Remove(fitLink)

		err = os.Link(path.Join(p.Dir, "bsp", fitFile), fitLink)
		if err != nil {
			return err
		}
	} else {
		kernelLink := path.Join(p.Dir, ".pv", "pv-kernel.img")
		initrdLink := path.Join(p.Dir, ".pv", "pv-initrd.img")
		dtbLink := ""
		if dtbFile != "" {
			dtbLink = path.Join(p.Dir, ".pv", "pv-fdt.dtb")
		}
		os.Mkdir(path.Join(p.Dir, ".pv"), 0755)
		os.Remove(kernelLink)
		os.Remove(initrdLink)
		if dtbLink != "" {
			os.Remove(dtbLink)
		}

		err = os.Link(path.Join(p.Dir, "bsp", kernelFile), kernelLink)
		if err != nil {
			return err
		}
		err = os.Link(path.Join(p.Dir, "bsp", initrdFile), initrdLink)
		if err != nil {
			return err
		}

		if dtbFile != "" {
			err = os.Link(path.Join(p.Dir, "bsp", dtbFile), dtbLink)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
