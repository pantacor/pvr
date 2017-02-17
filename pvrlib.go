/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"pvr/api"

	"github.com/asac/json-patch"
	"github.com/go-resty/resty"
	"github.com/urfave/cli"
)

type PvrStatus struct {
	NewFiles     []string
	RemovedFiles []string
	ChangedFiles []string
	JsonDiff     *[]byte
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
	return str
}

type PvrMap map[string]interface{}
type PvrIndex map[string]string

type Pvr struct {
	Initialized     bool
	Dir             string
	Pvrdir          string
	Objdir          string
	PristineJson    []byte
	PristineJsonMap PvrMap
	NewFiles        PvrIndex
	App             *cli.App
}

func (p *Pvr) String() string {
	return "PVR: " + p.Dir
}

func NewPvr(app *cli.App, dir string) (*Pvr, error) {
	pvr := Pvr{
		Dir:         dir + "/",
		Pvrdir:      path.Join(dir, ".pvr"),
		Objdir:      path.Join(dir, ".pvr", "objects"),
		Initialized: false,
		App:         app,
	}

	fileInfo, err := os.Stat(pvr.Dir)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("pvr path is not a directory: " + dir)
	}

	fileInfo, err = os.Stat(path.Join(pvr.Pvrdir, "json"))

	if err == nil && fileInfo.IsDir() {
		return nil, errors.New("Repo is in bad state. .pvr/json is a directory")
	}
	if err != nil {
		pvr.Initialized = false
		return &pvr, nil
	}

	byteJson, err := ioutil.ReadFile(path.Join(pvr.Pvrdir, "json"))
	// pristine json we keep as string as this will allow users load into
	// convenient structs
	pvr.PristineJson = byteJson

	err = json.Unmarshal(pvr.PristineJson, &pvr.PristineJsonMap)
	if err != nil {
		return nil, errors.New("JSON Unmarshal (" + strings.TrimPrefix(path.Join(pvr.Pvrdir, "json"), pvr.Dir) + "): " + err.Error())
	}

	// new files is a json file we will parse happily
	bytesNew, err := ioutil.ReadFile(path.Join(pvr.Pvrdir, "new"))
	if err == nil {
		err = json.Unmarshal(bytesNew, &pvr.NewFiles)
	} else {
		pvr.NewFiles = map[string]string{}
		err = nil
	}

	if err != nil {
		return &pvr, errors.New("Repo in bad state. JSON Unmarshal (" + strings.TrimPrefix(path.Join(pvr.Pvrdir, "json"), pvr.Dir) + ") Not possible. Make a copy of the repository for forensics, file a bug and maybe delete that file manually to try to recover: " + err.Error())
	}

	return &pvr, nil
}

func (p *Pvr) addPvrFile(path string) error {
	shaBal, err := FiletoSha(path)
	if err != nil {
		return err
	}
	relPath := strings.TrimPrefix(path, p.Dir)
	p.NewFiles[relPath] = shaBal
	return nil
}

// XXX: make this git style
func (p *Pvr) AddFile(globs []string) error {

	err := filepath.Walk(p.Dir, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(walkPath, p.Pvrdir) {
			return nil
		}

		// no globs specified: add all
		if len(globs) == 0 || (len(globs) == 1 && globs[0] == ".") {
			p.addPvrFile(walkPath)
		}
		for _, glob := range globs {
			absglob := glob
			if absglob[0] != '/' {
				absglob = p.Dir + glob
			}
			matched, err := filepath.Match(absglob, walkPath)
			if err != nil {
				fmt.Println("WARNING: cannot read file (" + err.Error() + "):" + walkPath)
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

	jsonData, err := json.Marshal(p.NewFiles)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(p.Pvrdir, "new.XXX"), jsonData, 0644)
	if err != nil {
		return err
	}
	err = os.Rename(path.Join(p.Pvrdir, "new.XXX"), path.Join(p.Pvrdir, "new"))
	if err != nil {
		return err
	}
	return nil
}

// create the canonical json for the working directory
func (p *Pvr) GetWorkingJson() ([]byte, error) {

	workingJson := map[string]interface{}{}
	workingJson["#spec"] = "pantavisor-multi-platform@1"

	err := filepath.Walk(p.Dir, func(filePath string, info os.FileInfo, err error) error {
		relPath := strings.TrimPrefix(filePath, p.Dir)
		// ignore .pvr directory
		if _, ok := p.PristineJsonMap[relPath]; !ok {
			if _, ok1 := p.NewFiles[relPath]; !ok1 {
				return nil
			}
		}
		if info.IsDir() {
			return nil
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
			workingJson[relPath] = jsonFile
		} else {
			sha, err := FiletoSha(filePath)
			if err != nil {
				return err
			}
			workingJson[relPath] = sha
		}

		return nil
	})

	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(workingJson)
}

func (p *Pvr) Init() error {

	return p.InitCustom("")
}

func (p *Pvr) InitCustom(customInitJson string) error {

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
	err = os.Mkdir(p.Objdir, 0755)

	jsonFile, err := os.OpenFile(path.Join(p.Pvrdir, "json"), os.O_CREATE|os.O_WRONLY, 0644)

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
	workingJson, err := p.GetWorkingJson()
	if err != nil {
		return nil, err
	}

	diff, err := jsonpatch.CreateMergePatch(p.PristineJson, workingJson)
	return &diff, nil
}

func (p *Pvr) Status() (*PvrStatus, error) {
	rs := PvrStatus{}

	// produce diff of working dir to prisitine
	diff, err := p.Diff()
	if err != nil {
		return nil, err
	}
	rs.JsonDiff = diff

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

	return &rs, nil
}

func (p *Pvr) Commit(msg string) error {
	status, err := p.Status()

	if err != nil {
		return err
	}

	for _, v := range status.ChangedFiles {
		fmt.Println("Committing " + v)
		if strings.HasSuffix(v, ".json") {
			continue
		}
		sha, err := FiletoSha(v)
		if err != nil {
			return err
		}
		err = Copy(path.Join(p.Objdir, sha), v)
		if err != nil {
			return err
		}
	}

	// copy all objects with atomic commit
	for _, v := range status.NewFiles {
		fmt.Println("Adding " + v)
		if strings.HasSuffix(v, ".json") {
			continue
		}
		sha, err := FiletoSha(v)
		if err != nil {
			return err
		}
		_, err = os.Stat(path.Join(p.Objdir, sha))
		// if not exists, then copy; otherwise continue
		if err != nil {

			err = Copy(path.Join(p.Objdir, sha+".new"), v)
			if err != nil {
				return err
			}
			err = os.Rename(path.Join(p.Objdir, sha+".new"),
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
		fmt.Println("Removing " + v)
	}

	ioutil.WriteFile(path.Join(p.Pvrdir, "commitmsg.new"), []byte(msg), 0644)
	err = os.Rename(path.Join(p.Pvrdir, "commitmsg.new"), path.Join(p.Pvrdir, "commitmsg"))
	if err != nil {
		return err
	}

	newJson, err := jsonpatch.MergePatch(p.PristineJson, *status.JsonDiff)

	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path.Join(p.Pvrdir, "json.new"), newJson, 0644)

	if err != nil {
		return err
	}

	err = os.Rename(path.Join(p.Pvrdir, "json.new"), path.Join(p.Pvrdir, "json"))

	if err != nil {
		return err
	}

	// ignore error here as new might not exist
	os.Remove(path.Join(p.Pvrdir, "new"))

	return nil
}

func (p *Pvr) PutLocal(repoPath string) error {

	_, err := os.Stat(repoPath)
	if err != os.ErrNotExist {
		err = os.MkdirAll(repoPath, 0755)
	}
	if err != nil {
		return err
	}

	objectsPath := path.Join(repoPath, "objects")
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
		Copy(path.Join(objectsPath, v)+".new", path.Join(p.Dir, ".pvr", v))

	}
	err = filepath.Walk(p.Objdir, func(filePath string, info os.FileInfo, err error) error {
		// ignore directories
		if info.IsDir() {
			return nil
		}
		base := path.Base(filePath)
		err = Copy(path.Join(objectsPath, base+".new"), filePath)
		if err != nil {
			return err
		}

		err = os.Rename(path.Join(objectsPath, base+".new"),
			path.Join(objectsPath, base))
		return err
	})

	err = Copy(path.Join(repoPath, "json.new"), path.Join(p.Pvrdir, "json"))
	if err != nil {
		return err
	}

	return os.Rename(path.Join(repoPath, "json.new"),
		path.Join(repoPath, "json"))
}

type PvrInfo struct {
	jsonUrl string `json:json-url`
	objUrl  string `json:object-url`
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

func (p *Pvr) initializeRemote(repoPath string) (pvrapi.PvrRemote, error) {
	res := pvrapi.PvrRemote{}
	repoUrl, err := url.Parse(repoPath)

	if err != nil {
		return res, err
	}

	pvrRemoteUrl := repoUrl
	pvrRemoteUrl.Path = path.Join(pvrRemoteUrl.Path, ".pvrremote")

	response, err := resty.R().
		SetAuthToken(p.App.Metadata["PANTAHUB_AUTH"].(string)).
		Get(pvrRemoteUrl.String())

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
func (p *Pvr) listFilesAndObjects() (map[string]string, error) {

	filesAndObjects := map[string]string{}
	// push all objects
	for k, v := range p.PristineJsonMap {
		if strings.HasSuffix(k, ".json") {
			continue
		}
		if strings.HasPrefix(k, "#spec") {
			continue
		}
		objId, ok := v.(string)

		if !ok {
			return map[string]string{}, errors.New("bad object id for file '" + k + "' in pristine pvr json")
		}
		filesAndObjects[k] = objId
	}
	return filesAndObjects, nil
}

func (p *Pvr) postObjects(pvrRemote pvrapi.PvrRemote) error {

	filesAndObjects, err := p.listFilesAndObjects()
	if err != nil {
		return err
	}

	// push all objects
	for k, v := range filesAndObjects {
		info, err := os.Stat(path.Join(p.Objdir, v))
		if err != nil {
			return err
		}
		sizeString := fmt.Sprintf("%d", info.Size)

		remoteObject := ObjectWithAccess{}
		remoteObject.Object.Size = sizeString
		remoteObject.MimeType = "application/octet-stream"
		remoteObject.Sha = v
		remoteObject.ObjectName = k

		response, err := resty.R().SetBody(remoteObject).
			SetAuthToken(p.App.Metadata["PANTAHUB_AUTH"].(string)).
			Post(pvrRemote.ObjectsEndpointUrl + "/")

		if err != nil {
			return err
		}

		err = json.Unmarshal(response.Body(), &remoteObject)
		if err != nil {
			return err
		}

		fmt.Print("Uploading object to " + remoteObject.SignedPutUrl)

		if err != nil {
			return err
		}

		fileName := path.Join(p.Objdir, v)
		fileBytes, _ := ioutil.ReadFile(fileName)
		response, err = resty.R().SetBody(fileBytes).SetContentLength(true).Put(remoteObject.SignedPutUrl)

		if err != nil {
			return err
		}

		fmt.Println("Upload done.")

		if 200 != response.StatusCode() {
			return errors.New("REST call failed. " +
				strconv.Itoa(response.StatusCode()) + "  " + response.Status())

		}
	}
	return nil
}

func (p *Pvr) PutRemote(repoPath string) error {

	pvrRemote, err := p.initializeRemote(repoPath)

	if err != nil {
		return err
	}

	err = p.postObjects(pvrRemote)

	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(path.Join(p.Pvrdir, "json"))

	if err != nil {
		return err
	}

	body := map[string]interface{}{}
	json.Unmarshal(data, &body)

	response, err := resty.R().SetBody(body).
		SetAuthToken(p.App.Metadata["PANTAHUB_AUTH"].(string)).
		Put(pvrRemote.JsonGetUrl)

	if err != nil {
		return err
	}

	if 200 != response.StatusCode() {
		return errors.New("REST call failed. " +
			strconv.Itoa(response.StatusCode()) + "  " + response.Status())

	}

	err = json.Unmarshal(response.Body(), &body)

	if err != nil {
		return err
	}

	return nil
}

func (p *Pvr) Put(uri string) error {
	url, err := url.Parse(uri)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		return p.PutLocal(uri)
	}

	return p.PutRemote(uri)
}

func (p *Pvr) PutObjects(uri string) error {
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

	return p.postObjects(pvr)
}

// make a json post to a REST endpoint. You can provide metainfo etc. in post
// argument as json. postKey if set will be used as key that refers to the posted
// json. Example usage: json blog post, json revision repo with commit message etc
func (p *Pvr) Post(uri string, envelope string) error {
	url, err := url.Parse(uri)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		return errors.New("Post must be a remote REST endpoint, not: " + url.String())
	}

	remotePvr, err := p.initializeRemote(uri)

	if err != nil {
		return err
	}

	err = p.postObjects(remotePvr)

	if err != nil {
		return err
	}

	envJson := map[string]interface{}{}
	err = json.Unmarshal([]byte(envelope), &envJson)

	if err != nil {
		return err
	}

	if remotePvr.JsonKey != "" {
		envJson[remotePvr.JsonKey] = p.PristineJsonMap
	} else {
		envJson["post"] = p.PristineJsonMap
	}

	data, err := json.Marshal(envJson)

	if err != nil {
		return err
	}

	response, err := resty.R().SetBody(data).SetContentLength(true).
		SetAuthToken(p.App.Metadata["PANTAHUB_AUTH"].(string)).
		Post(remotePvr.PostUrl)

	if err != nil {
		return err
	}

	if response.StatusCode() != 200 {
		return errors.New("REST call failed. " +
			strconv.Itoa(response.StatusCode()) + "  " + response.Status())
	}

	fmt.Println("Posted JSON: " + string(response.Body()))

	return nil
}

func (p *Pvr) GetRepoLocal(repoPath string) error {

	// first copy new json, but only rename at the very end after all else succeed
	jsonNew := path.Join(p.Pvrdir, "json.new")
	err := Copy(jsonNew, path.Join(repoPath, "json"))
	rs := map[string]interface{}{}

	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(jsonNew)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &rs)
	if err != nil {
		return errors.New("JSON Unmarshal (" +
			strings.TrimPrefix(jsonNew, p.Dir) + "): " + err.Error())
	}

	for k, v := range rs {
		if strings.HasSuffix(k, ".json") {
			continue
		}
		if strings.HasPrefix(k, "#spec") {
			continue
		}
		getPath := path.Join(repoPath, "objects", v.(string))
		objPathNew := path.Join(p.Objdir, v.(string)+".new")
		objPath := path.Join(p.Objdir, v.(string))
		fmt.Println("pulling objects file " + getPath + "-> " + objPathNew)
		err := Copy(objPathNew, getPath)
		if err != nil {
			return err
		}
		err = os.Rename(objPathNew, objPath)
		if err != nil {
			return err
		}
	}

	// all succeeded, atomically commiting the json
	err = os.Rename(jsonNew, strings.TrimSuffix(jsonNew, ".new"))

	return err
}

func (p *Pvr) getObjects(pvrRemote pvrapi.PvrRemote) error {

	// push all objects
	response, err := resty.R().
		SetAuthToken(p.App.Metadata["PANTAHUB_AUTH"].(string)).
		Get(pvrRemote.JsonGetUrl)

	if err != nil {
		return err
	}

	jsonNew := response.Body()
	jsonMap := map[string]interface{}{}

	err = json.Unmarshal(response.Body(), &jsonMap)

	for k := range jsonMap {
		if strings.HasSuffix(k, ".json") {
			continue
		}
		if strings.HasPrefix(k, "#spec") {
			continue
		}
		v := jsonMap[k].(string)

		response, err := resty.R().
			SetAuthToken(p.App.Metadata["PANTAHUB_AUTH"].(string)).
			Get(pvrRemote.ObjectsEndpointUrl + "/" + v)

		if err != nil {
			return err
		}

		if response.StatusCode() != 200 {
			return errors.New("REST call failed. " +
				strconv.Itoa(response.StatusCode()) + "  " + response.Status())
		}

		remoteObject := ObjectWithAccess{}
		err = json.Unmarshal(response.Body(), &remoteObject)

		if err != nil {
			return err
		}

		response, err = resty.R().Get(remoteObject.SignedGetUrl)

		if err != nil {
			return err
		}
		if response.StatusCode() != 200 {
			return errors.New("REST call failed. " +
				strconv.Itoa(response.StatusCode()) + "  " + response.Status())
		}

		ioutil.WriteFile(path.Join(p.Objdir, v), response.Body(), 0644)
		fmt.Println("Downloaded Object " + v)
	}
	err = ioutil.WriteFile(path.Join(p.Pvrdir, "json.new"), jsonNew, 0644)

	if err != nil {
		return err
	}

	return os.Rename(path.Join(p.Pvrdir, "json.new"), path.Join(p.Pvrdir, "json"))
}

func (p *Pvr) GetRepoRemote(repoPath string) error {

	url, err := url.Parse(repoPath)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		return errors.New("Post must be a remote REST endpoint, not: " + url.String())
	}

	remotePvr, err := p.initializeRemote(repoPath)

	if err != nil {
		return err
	}

	err = p.getObjects(remotePvr)

	if err != nil {
		return err
	}

	return nil
}

func (p *Pvr) GetRepo(repoPath string) error {
	url, err := url.Parse(repoPath)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		return p.GetRepoLocal(repoPath)
	}

	return p.GetRepoRemote(repoPath)
}

func (p *Pvr) Reset() error {
	data, err := ioutil.ReadFile(path.Join(p.Pvrdir, "json"))

	if err != nil {
		return err
	}
	jsonMap := map[string]interface{}{}

	err = json.Unmarshal(data, &jsonMap)

	if err != nil {
		return errors.New("JSON Unmarshal (" +
			strings.TrimPrefix(path.Join(p.Pvrdir, "json"), p.Dir) + "): " +
			err.Error())
	}

	for k, v := range jsonMap {
		if strings.HasSuffix(k, ".json") {
			data, err := json.Marshal(v)
			if err != nil {
				return err
			}
			err = ioutil.WriteFile(path.Join(p.Dir, k+".new"), data, 0644)
			if err != nil {
				return err
			}
			err = os.Rename(path.Join(p.Dir, k+".new"),
				path.Join(p.Dir, k))

		} else if strings.HasPrefix(k, "#spec") {
			continue
		} else {
			objectP := path.Join(p.Objdir, v.(string))
			targetP := path.Join(p.Dir, k)
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

			err = Copy(targetP+".new", objectP)
			if err != nil {
				return err
			}
			err = os.Rename(targetP+".new", targetP)
			if err != nil {
				return err
			}
		}
	}
	os.Remove(path.Join(p.Pvrdir, "new"))
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

func (p *Pvr) Export(dst string) error {

	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()

	var fileWriter io.WriteCloser

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

	filesAndObjects, err := p.listFilesAndObjects()
	if err != nil {
		return err
	}

	for _, v := range filesAndObjects {
		apath := "objects/" + v
		ipath := path.Join(p.Objdir, v)
		err := addToTar(tw, apath, ipath)

		if err != nil {
			return err
		}
	}

	if err := addToTar(tw, "json", path.Join(p.Pvrdir, "json")); err != nil {
		return err
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

		filePath := path.Join(p.Pvrdir, header.Name)
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
