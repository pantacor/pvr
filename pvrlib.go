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
		Dir:    dir,
		Pvrdir: path.Join(dir, ".pvr"),
		Objdir: path.Join(dir, ".pvr", "objects"),
	}
	fileInfo, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !fileInfo.IsDir() {
		return nil, errors.New("pvr path is not a directory: " + dir)
	}
	pvr.Dir += "/"

	// trim all double /
	tmp := strings.TrimSuffix(pvr.Dir, "//")
	for ; tmp != pvr.Dir; tmp = strings.TrimSuffix(pvr.Dir, "//") {
	}

	byteJson, err := ioutil.ReadFile(dir + "/.pvr/json")
	// pristine json we keep as string as this will allow users load into
	// convenient structs
	pvr.PristineJson = byteJson

	err = json.Unmarshal(pvr.PristineJson, &pvr.PristineJsonMap)
	if err != nil {
		return nil, err
	}

	// new files is a json file we will parse happily
	bytesNew, err := ioutil.ReadFile(dir + "/.pvr/new")
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

	err := filepath.Walk(p.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(path, p.Dir+".pvr") {
			return nil
		}

		// no globs specified: add all
		if len(globs) == 0 || (len(globs) == 1 && globs[0] == ".") {
			p.addPvrFile(path)
		}
		for _, glob := range globs {
			absglob := glob
			if absglob[0] != '/' {
				absglob = p.Dir + glob
			}
			matched, err := filepath.Match(absglob, path)
			if err != nil {
				fmt.Println("WARNING: cannot read file (" + err.Error() + "):" + path)
				return err
			}
			if matched {
				err = p.addPvrFile(path)
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

	err = ioutil.WriteFile(p.Dir+"/.pvr/new.XXX", jsonData, 0644)
	if err != nil {
		return err
	}
	err = os.Rename(p.Dir+"/.pvr/new.XXX", p.Dir+"/.pvr/new")
	if err != nil {
		return err
	}
	return nil
}

// create the canonical json for the working directory
func (p *Pvr) GetWorkingJson() ([]byte, error) {

	workingJson := map[string]interface{}{}
	workingJson["#spec"] = "pantavisor-multi-platform@1"

	err := filepath.Walk(p.Dir, func(path string, info os.FileInfo, err error) error {
		relPath := strings.TrimPrefix(path, p.Dir)
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
		if strings.HasSuffix(filepath.Base(path), ".json") {
			jsonFile := map[string]interface{}{}

			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			err = json.Unmarshal(data, &jsonFile)
			if err != nil {
				return err
			}
			workingJson[relPath] = jsonFile
		} else {
			sha, err := FiletoSha(path)
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

	ioutil.WriteFile(p.Dir+".pvr/commitmsg.new", []byte(msg), 0644)
	err = os.Rename(p.Dir+".pvr/commitmsg.new", p.Dir+".pvr/commitmsg")
	if err != nil {
		return err
	}

	newJson, err := jsonpatch.MergePatch(p.PristineJson, *status.JsonDiff)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(p.Dir+".pvr/json.new", newJson, 0644)

	if err != nil {
		return err
	}

	err = os.Rename(p.Dir+".pvr/json.new", p.Dir+".pvr/json")

	if err != nil {
		return err
	}

	err = os.Remove(path.Join(p.Pvrdir, "new"))

	return err
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
		return err
	}

	for k, v := range rs {
		if strings.HasSuffix(k, ".json") {
			continue
		}
		if strings.HasPrefix(k, "#spec") {
			continue
		}
		getPath := path.Join(repoPath, "objects", v.(string))
		objPathNew := path.Join(p.Pvrdir, v.(string)+".new")
		objPath := path.Join(p.Pvrdir, v.(string))
		fmt.Println("pulling objects file " + getPath)
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
