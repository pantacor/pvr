/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evanphx/json-patch"
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
}

func (p *Pvr) String() string {
	return "PVR: " + p.Dir
}

func NewPvr(dir string) (*Pvr, error) {
	pvr := &Pvr{
		Dir:         path.Join(dir),
		Pvrdir:      path.Join(dir, ".pvr"),
		Objdir:      path.Join(dir, ".pvr", "objects"),
		Initialized: false,
	}
	fileInfo, err := os.Stat(pvr.Dir)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("pvr path is not a directory: " + dir)
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
	}
	return pvr, nil
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
	err = ioutil.WriteFile(path.Join(p.Pvrdir, ".pvr/json.new"), newJson, 0644)

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

func (p *Pvr) PushLocal(repoPath string) error {

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

func (p *Pvr) PushRemote(repoPath string) error {
	return errors.New("Not Implemented")
}

func (p *Pvr) Push(uri string) error {
	url, err := url.Parse(uri)

	if err != nil {
		return err
	}

	if url.Scheme == "" {
		return p.PushLocal(uri)
	}

	return p.PushRemote(uri)
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
		return errors.New("JSON Unmarshal (" + strings.TrimPrefix(jsonNew, p.Dir) + "): " + err.Error())
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

func (p *Pvr) GetRepoRemote(repoPath string) error {
	return errors.New("Not Implemented.")
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
		return errors.New("JSON Unmarshal (" + strings.TrimPrefix(path.Join(p.Pvrdir, "json"), p.Dir) + "): " + err.Error())
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
