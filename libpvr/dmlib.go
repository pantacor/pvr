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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	cjson "github.com/gibson042/canonicaljson-go"
	"gitlab.com/pantacor/pvr/utils/pvjson"
)

const (
	DmVolumes = "_dm"
)

type DmSource struct {
	Source
	DmVolumes []string `json:"dm_volumes,omitempty"`
}

type DmVerityJson struct {
	DataDevice string `json:"data_device"`
	HashDevice string `json:"hash_device"`
	RootHash   string `json:"root_hash"`
}

func (p *Pvr) dmifySrcJson(container, volume string) error {

	appManifest, err := p.GetApplicationManifest(container)

	if err != nil {
		return err
	}

	if appManifest.DmEnabled == nil {
		appManifest.DmEnabled = map[string]bool{}
	}

	appManifest.DmEnabled[volume] = true

	srcContent, err := json.MarshalIndent(appManifest, " ", " ")
	if err != nil {
		return err
	}

	srcFilePath := filepath.Join(p.Dir, container, SRC_FILE)
	err = os.WriteFile(srcFilePath, srcContent, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pvr) dmifyRunJson(container, volume string) error {
	var runJson map[string]interface{}
	var itmp interface{}

	runJsonPath := path.Join(p.Dir, container, "run.json")
	fBuf, err := os.ReadFile(runJsonPath)
	if err != nil {
		return err
	}
	pvjson.Unmarshal(fBuf, &runJson)

	if runJson["modules"] != nil && runJson["modules"].(string) == "dm:"+volume {
		// do nothing
	} else if runJson["modules"] != nil && runJson["modules"].(string) == volume {
		runJson["modules"] = "dm:" + volume
	} else if runJson["firmware"] != nil && runJson["firmware"].(string) == "dm:"+volume {
		// do nothing
	} else if runJson["firmware"] != nil && runJson["firmware"].(string) == volume {
		runJson["firmware"] = "dm:" + volume
	} else if runJson["root-volume"] != nil && runJson["root-volume"].(string) == "dm:"+volume {
		// do nothing
	} else if runJson["root-volume"] != nil && runJson["root-volume"].(string) == volume {
		runJson["root-volume"] = "dm:" + volume
	} else if runJson["volumes"] != nil {
		itmp = runJson["volumes"]
		volumes := itmp.([]interface{})
		newVolumes := []string{}
		for _, v := range volumes {
			vS := v.(string)
			if vS == volume {
				newVolumes = append(newVolumes, "dm:"+volume)
			} else {
				newVolumes = append(newVolumes, v.(string))
			}
		}
		runJson["volumes"] = newVolumes
	} else {
		return errors.New("volume to dmify not found: " + volume)
	}

	outB, err := cjson.Marshal(runJson)
	if err != nil {
		return err
	}

	err = os.WriteFile(runJsonPath+".new", outB, 0644)
	if err != nil {
		return err
	}
	err = os.Rename(runJsonPath+".new", runJsonPath)
	if err != nil {
		return err
	}
	fmt.Println("- Updated " + runJsonPath)

	return nil
}

func verityOutputParse(outS string, vm map[string]interface{}) error {
	var tmpString string
	idx := strings.Index(outS, "Root hash:")
	if idx < 0 {
		return errors.New("no root hash found in out: " + outS)
	}

	idx2 := strings.Index(outS[idx+10:], "\n")
	if idx2 < 0 {
		tmpString = outS[idx+10:]
	} else {
		tmpString = outS[idx+10 : idx+10+idx2]
	}
	vm["root_hash"] = strings.Trim(tmpString, " \t")

	idx = strings.Index(outS, "UUID:")
	if idx < 0 {
		return errors.New("no UUID found in out: " + outS)
	}
	idx2 = strings.Index(outS[idx+5:], "\n")
	if idx2 < 0 {
		tmpString = outS[idx+5:]
	} else {
		tmpString = outS[idx+5 : idx+5+idx2]
	}
	vm["uuid"] = strings.Trim(tmpString, " \t")

	idx = strings.Index(outS, "Salt:")
	if idx < 0 {
		return errors.New("no Salt found in out: " + outS)
	}
	idx2 = strings.Index(outS[idx+5:], "\n")
	if idx2 < 0 {
		tmpString = outS[idx+5:]
	} else {
		tmpString = outS[idx+5 : idx+5+idx2]
	}
	vm["salt"] = strings.Trim(tmpString, " \t")

	return nil
}

func (p *Pvr) DmCVerityApply(prefix string) error {

	workingJson, _, err := p.GetWorkingJsonMap()
	if err != nil {
		return err
	}

	for k, v := range workingJson {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		idx := strings.Index(k, "/"+DmVolumes+"/")
		if idx > 0 {
			var dataDevice string
			var hashDevice string

			container := k[:idx]
			volume := k[idx+len("/"+DmVolumes+"/"):]
			volume = strings.TrimSuffix(volume, ".json")

			vm := v.(map[string]interface{})
			vm["type"] = "dm-verity"
			dataDevice = vm["data_device"].(string)
			hashDevice = vm["hash_device"].(string)

			flags := []string{}

			flags = append(flags, "format")
			if vm["salt"] != nil {
				flags = append(flags, "--salt", vm["salt"].(string))
			}
			if vm["uuid"] != nil {
				flags = append(flags, "--uuid", vm["uuid"].(string))
			}
			flags = append(flags, dataDevice)
			flags = append(flags, hashDevice)
			cmd := exec.Command("veritysetup", flags...)
			cmd.Dir = path.Join(p.Dir, container)
			outPipe, err := cmd.StdoutPipe()
			if err != nil {
				return err
			}
			err = cmd.Start()

			if err != nil {
				return err
			}
			out, err := io.ReadAll(outPipe)
			if err != nil {
				return err
			}

			manifestPath := path.Join(p.Dir, container, DmVolumes, volume+".json")

			if len(out) == 0 {
				fmt.Println("- Unchanged verity format " + manifestPath)
				return nil
			}
			outS := string(out)

			err = verityOutputParse(outS, vm)

			if err != nil {
				return err
			}

			outB, err := cjson.Marshal(vm)
			if err != nil {
				return err
			}

			err = os.MkdirAll(path.Dir(manifestPath), 0755)
			if err != nil {
				return err
			}

			err = os.WriteFile(manifestPath+".new", outB, 0644)
			if err != nil {
				return err
			}
			os.Rename(manifestPath+".new", manifestPath)
			p.AddFile([]string{path.Join(container, hashDevice)}, false)

			fmt.Println("- Updated " + manifestPath)

		}
	}
	return nil
}

func (p *Pvr) DmCVerityConvert(container string, volume string) error {

	fmt.Printf("container=%s volume=%s\n", container, volume)
	manifestPath := path.Join(p.Dir, container, DmVolumes, volume+".json")

	var manifestMap map[string]interface{}

	_, err := os.Stat(manifestPath)

	if os.IsNotExist(err) {
		err = os.MkdirAll(path.Dir(manifestPath), 0755)
		if err != nil {
			return err
		}
		err = os.WriteFile(manifestPath, []byte("{}"), 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	buf, err := os.ReadFile(manifestPath)

	if err != nil {
		return err
	}
	err = pvjson.Unmarshal(buf, &manifestMap)

	if err != nil {
		return err
	}

	manifestMap["type"] = "dm-verity"
	manifestMap["data_device"] = volume
	manifestMap["hash_device"] = volume + ".hash"

	cmd := exec.Command("veritysetup", "format", manifestMap["data_device"].(string),
		manifestMap["hash_device"].(string))
	cmd.Dir = path.Join(p.Dir, container)
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()

	if err != nil {
		return err
	}
	out, err := io.ReadAll(outPipe)
	if err != nil {
		return err
	}
	outS := string(out)
	err = verityOutputParse(outS, manifestMap)
	if err != nil {
		return err
	}

	outB, err := cjson.Marshal(manifestMap)
	if err != nil {
		return err
	}
	err = os.WriteFile(manifestPath+".new", outB, 0644)
	if err != nil {
		return err
	}
	os.Rename(manifestPath+".new", manifestPath)
	p.AddFile([]string{path.Join(container, manifestMap["hash_device"].(string)),
		path.Join(container, DmVolumes, volume+".json")}, false)

	fmt.Println("- Updated " + manifestPath)

	// update run.json

	p.dmifyRunJson(container, volume)
	p.dmifySrcJson(container, volume)

	return nil
}

func (p *Pvr) DmCryptApply() error {
	return nil
}
