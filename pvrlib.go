/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Pvr struct {
	Dir             string
	PristineJson    string
	PristineJsonMap PvrMap
	NewFiles        PvrIndex
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
	pvr.PristineJson = string(byteJson)

	err = json.Unmarshal(byteJson, &pvr.PristineJsonMap)
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

func FiletoSha(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	// problems reading file here, just dont add, output warning
	if err != nil {
		return "", err
	}

	buf := sha256.Sum256(data)
	shaBal := hex.EncodeToString(buf[:])
	return shaBal, nil
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

func (p *Pvr) GetWorkingJson() ([]byte, error) {

	workingJson := map[string]interface{}{}

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
