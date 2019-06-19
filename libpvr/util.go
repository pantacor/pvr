//
// Copyright 2017  Pantacor Ltd.
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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func Copy(dst, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	cerr := out.Close()
	if err != nil {
		return err
	}
	return cerr
}

func FormatJson(data []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, data, "", "\t")
	if error != nil {
		return []byte(""), error
	}

	return prettyJSON.Bytes(), nil
}

func FiletoSha(path string) (string, error) {
	hasher := sha256.New()

	file, err := os.Open(path)
	// problems reading file here, just dont add, output warning
	if err != nil {
		return "", err
	}

	defer file.Close()

	_, err = io.Copy(hasher, file)

	if err != nil {
		return "", err
	}

	buf := hasher.Sum(nil)
	shaBal := hex.EncodeToString(buf[:])
	return shaBal, nil
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func GetPhAuthHeaderTokenKey(authHeader string) (string, error) {
	// no auth header; nothing we can do magic here...
	if authHeader == "" {
		return "", errors.New("Bad Parameter (authHeader empty)")
	}

	authType, opts := getWwwAuthenticateInfo(authHeader)
	if authType != "JWT" && authType != "Bearer" {
		return "", errors.New("Invalid www-authenticate header retrieved")
	}

	realm := opts["realm"]
	authEpString := opts["ph-aeps"]
	authEps := strings.Split(authEpString, ",")

	if len(authEps) == 0 || len(realm) == 0 {
		return "", errors.New("Bad Server Behaviour. Need ph-aeps and realm token in Www-Authenticate header. Check your server version")
	}

	return authEps[0] + " realm=" + realm, nil
}

// ReadOrCreateFile read a file from file system if is not avaible creates the file
func ReadOrCreateFile(filePath string) (*[]byte, error) {
	_, err := os.Stat(filePath)

	if os.IsNotExist(err) {
		_, err := os.Stat(filepath.Dir(filePath))
		if os.IsNotExist(err) {
			err = os.MkdirAll(filePath, 0700)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	}

	if err != nil {
		return nil, errors.New("OS error getting stats for: " + err.Error())
	}

	content, err := ioutil.ReadFile(filePath)

	if err != nil {
		return nil, errors.New("OS error reading file: " + err.Error())
	}

	return &content, nil
}

func WriteTxtFile(filePath string, content string) error {

	data := []byte(content)

	return ioutil.WriteFile(filePath, data, 0644)
}

// GetPlatform get string with the full platform name
func GetPlatform() string {
	values := []string{string(runtime.GOOS), string(runtime.GOARCH)}

	return strings.Join(values, "_")
}

// AskForConfirmation ask the user for confirmation action
func AskForConfirmation(question string) bool {
	var response string
	fmt.Println(question)

	_, err := fmt.Scanln(&response)
	if err != nil {
		return false
	}
	okayResponses := `(y|yes|Yes|Y|YES)`
	matched, err := regexp.MatchString(okayResponses, response)
	if err != nil {
		return false
	}
	return matched
}

func FileHasSameSha(path, sha string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	fileSha, err := FiletoSha(path)
	if err != nil {
		return false, err
	}

	fileSha = fmt.Sprintf("sha256:%s", fileSha)

	return fileSha == sha, nil
}

func CreateFolder(path string) error {
	_, err := os.Stat(path)

	if os.IsNotExist(err) {
		_, err := os.Stat(filepath.Dir(path))
		if os.IsNotExist(err) {
			err = os.MkdirAll(filepath.Dir(path), 0700)
			if err != nil {
				return err
			}
		}
		err = os.MkdirAll(path, 0700)
		if err != nil {
			return err
		}
	}
	return nil
}

func ExtractFiles(files []string, extractPath string) error {
	tarPath, err := exec.LookPath(TAR_CMD)
	if err != nil {
		return err
	}

	if tarPath == "" {
		return ErrTarNotFound
	}

	for _, file := range files {
		args := []string{tarPath, "xzvf", file, "-C", extractPath}
		untar := exec.Command(args[0], args[1:]...)
		untarError := untar.Run()
		if untarError != nil {
			return untarError
		}
	}

	return nil
}

func ReportError(err error, knowSolutions ...string) error {
	msg := "ERROR: "
	msg += err.Error()

	if len(knowSolutions) > 0 {
		msg += "\n  POSSIBLE SOLUTIONS:\n"
	}

	for i, solution := range knowSolutions {
		msg += fmt.Sprintf("   %d. %s\n", i+1, solution)
	}

	return errors.New(msg)
}
