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
package libpvr

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
)

const (
	defaultDistributionTag = "develop"
	defaultSpec            = "1"
)

// PvrGlobalConfig define all the posible general configuration for pvr
type PvrGlobalConfig struct {
	Spec            string `json:"Spec"`
	AutoUpgrade     bool   `json:"AutoUpgrade"`
	DistributionTag string `json:"DistributionTag"`
}

// LoadConfiguration read configuration from ~/.pvr/config.json or return default configuration
func LoadConfiguration(filePath string) (*PvrGlobalConfig, error) {
	fileContent, err := ReadOrCreateFile(filePath)
	if err != nil {
		return nil, errors.New("OS error getting stats for file in LoadConfiguration: " + err.Error())
	}

	config := defaultConfiguration()

	if fileContent == nil {
		err = WriteConfiguration(filePath, &config)
		if err != nil {
			return nil, errors.New("OS error writing LoadConfiguration: " + err.Error())
		}
	} else {
		err = json.Unmarshal(*fileContent, &config)
		if err != nil {
			return nil, errors.New("JSON Unmarshal error parsing config file in LoadConfiguration (" + filePath + "): " + err.Error())
		}
	}

	return &config, nil
}

// WriteConfiguration write ~/.pvr/config.json with new data
func WriteConfiguration(filePath string, config *PvrGlobalConfig) error {
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		_, err := os.Stat(filepath.Dir(filePath))
		if os.IsNotExist(err) {
			err = os.MkdirAll(filepath.Dir(filePath), 0700)
			if err != nil {
				return err
			}
		}
	}

	configurationJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return WriteTxtFile(filePath, string(configurationJSON))

}

// SetConfiguration read arguments and write new configuration, merging with the previous configuration on the file
func (pvr *Pvr) SetConfiguration(arguments []string) (*PvrGlobalConfig, error) {

	re := regexp.MustCompile(`^(.*)=(.*)$`)

	var configMap map[string]interface{}
	inrec, _ := json.Marshal(pvr.Session.Configuration)
	json.Unmarshal(inrec, &configMap)

	for _, variable := range arguments {
		value := re.FindStringSubmatch(variable)
		if len(value) == 3 {
			configMap[value[1]] = value[2]
		}
	}

	config := pvr.Session.Configuration
	err := fillStruct(configMap, config)
	if err != nil {
		return nil, err
	}

	configurationFilePath := filepath.Join(pvr.Session.configDir, ConfigurationFile)

	err = WriteConfiguration(configurationFilePath, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func defaultConfiguration() PvrGlobalConfig {
	return PvrGlobalConfig{
		Spec:            defaultSpec,
		AutoUpgrade:     true,
		DistributionTag: defaultDistributionTag,
	}
}

func fillStruct(data map[string]interface{}, result interface{}) error {
	structValue := reflect.ValueOf(result).Elem()
	for k, v := range data {
		structFieldValue := structValue.FieldByName(k)

		if structFieldValue.CanSet() && structFieldValue.IsValid() {
			if v == "true" {
				structFieldValue.Set(reflect.ValueOf(true))
			} else if v == "false" {
				structFieldValue.Set(reflect.ValueOf(false))
			} else {
				structFieldValue.Set(reflect.ValueOf(v))
			}

		}
	}
	return nil
}
