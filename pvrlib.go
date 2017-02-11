/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanphx/json-patch"
)

type Pvr struct {
	Dir             string
	PristineJson    []byte
	PristineJsonMap PvrMap
	NewFiles        PvrIndex
}

type PvrStatus struct {
	NewFiles     []string
	RemovedFiles []string
	ChangedFiles []string
	JsonDiff     *[]byte
}

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

func (p Pvr) String() string {
	return "PVR: " + p.Dir
}

func NewPvr(dir string) (*Pvr, error) {
	pvr := &Pvr{
		Dir: dir,
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
		sha, err := FiletoSha(v)
		if err != nil {
			return err
		}
		err = Copy(p.Dir+".pvr/objects/"+sha, v)
		if err != nil {
			return err
		}
	}

	for _, v := range status.NewFiles {
		sha, err := FiletoSha(v)
		if err != nil {
			return err
		}
		_, err = os.Stat(p.Dir + ".pvr/objects/" + sha)
		// if not exists, then copy; otherwise continue
		if err != nil {

			err = Copy(p.Dir+".pvr/objects/"+sha+".new", v)
			if err != nil {
				return err
			}
			err = os.Rename(p.Dir+".pvr/objects/"+sha+".new",
				p.Dir+".pvr/objects/"+sha)
			if err != nil {
				return err
			}
		}
		if err != nil {
			return err
		}
	}

	ioutil.WriteFile(p.Dir+".pvr/commitmsg", []byte(msg), 0644)

	newJson, err := jsonpatch.MergePatch(p.PristineJson, *status.JsonDiff)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(p.Dir+".pvr/json.new", newJson, 0644)

	if err != nil {
		return err
	}

	err = os.Rename(p.Dir+".pvr/json.new", p.Dir+".pvr/json")

	return err
}
