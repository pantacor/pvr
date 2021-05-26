//
// Copyright 2017-2020  Pantacor Ltd.
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
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/go-resty/resty"
)

// Variable set in main() to enable/disable debug
var IsDebugEnabled bool

// PrintDebugf forwards to Printfs if IsDebugEnabled
func PrintDebugf(format string, a ...interface{}) (n int, err error) {
	if IsDebugEnabled {
		return fmt.Fprintf(os.Stderr, format, a...)
	}
	return 0, nil
}

// PrintDebugln forwards to Printfs if IsDebugEnabled
func PrintDebugln(a ...interface{}) (n int, err error) {
	if IsDebugEnabled {
		return fmt.Fprintln(os.Stderr, a...)
	}
	return 0, nil
}

func Create(path string) error {
	touch, err := exec.LookPath(TOUCH_CMD)
	if err != nil {
		return err
	}
	args := []string{touch, path}
	touchCmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	touchCmd.Stdout = &out
	touchCmd.Stderr = &stderr
	err = touchCmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprint(err)+": "+stderr.String())
	}
	return nil
}

func MkdirAll(path string, perm os.FileMode) error {
	mkdir, err := exec.LookPath(MKDIR_CMD)
	if err != nil {
		return err
	}
	args := []string{mkdir, "-m", perm.String(), "-p", path}
	mkdirCmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	mkdirCmd.Stdout = &out
	mkdirCmd.Stderr = &stderr
	err = mkdirCmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprint(err)+": "+stderr.String())
	}
	return nil
}

func Mkdir(path string, perm os.FileMode) error {
	mkdir, err := exec.LookPath(MKDIR_CMD)
	if err != nil {
		return err
	}
	args := []string{mkdir, "-m", perm.String(), path}
	mkdirCmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	mkdirCmd.Stdout = &out
	mkdirCmd.Stderr = &stderr
	err = mkdirCmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprint(err)+": "+stderr.String())
	}
	return nil
}

// RemoveAll remove a path, could be a file or a folder
func RemoveAll(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	rm, err := exec.LookPath(RM_CMD)
	if err != nil {
		return err
	}
	args := []string{rm, "-rf", path}
	rmCmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	rmCmd.Stdout = &out
	rmCmd.Stderr = &stderr
	err = rmCmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprint(err)+": "+stderr.String())
	}
	return nil
}

// Remove remove a path, a file
func Remove(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	rm, err := exec.LookPath(RM_CMD)
	if err != nil {
		return err
	}
	args := []string{rm, "-f", path}
	rmCmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	rmCmd.Stdout = &out
	rmCmd.Stderr = &stderr
	err = rmCmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprint(err)+": "+stderr.String())
	}
	return nil
}

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

func Hardlink(dst, src string) error {
	os.Remove(dst)
	err := os.Link(src, dst)
	if err != nil {
		return err
	}
	err = os.Chmod(dst, 0444)
	return err
}

func RenameFile(src string, dst string) (err error) {
	err = Copy(dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy source file %s to %s: %s", src, dst, err)
	}
	err = os.RemoveAll(src)
	if err != nil {
		return fmt.Errorf("failed to cleanup source file %s: %s", src, err)
	}
	return nil
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

	if _, ok := opts["error"]; ok {
		return "", nil
	}

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
	arch := string(runtime.GOARCH)
	if arch == "arm" {
		arch = "armv6"
	}
	values := []string{string(runtime.GOOS), arch}

	return strings.Join(values, "_")
}

// AskForConfirmation ask the user for confirmation action
func AskForConfirmation(question string) bool {
	var response string
	fmt.Fprintln(os.Stderr, question)

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
		err := Untar(extractPath, file)
		if err != nil {
			return err
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

func StructToMap(s interface{}) (map[string]interface{}, error) {

	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}

	err = json.Unmarshal(b, &result)

	if err != nil {
		return nil, err
	}
	return result, nil
}

// CreateDevice : Create Device
func (s *Session) CreateDevice(baseURL string, deviceNick string) (
	*resty.Response,
	error,
) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	//Get User Access token
	host := u.Scheme + "://" + u.Host
	authHeader := "JWT realm=\"pantahub services\", ph-aeps=\"" + host + "/auth\""
	accessToken, err := s.auth.getCachedAccessToken(authHeader)
	if err != nil {
		return nil, err
	}
	response, err := s.DoAuthCall(func(req *resty.Request) (*resty.Response, error) {
		body := map[string]interface{}{}
		if deviceNick != "" {
			body["nick"] = deviceNick
		}
		return req.
			SetBody(body).
			SetAuthToken(accessToken).
			Post(baseURL + "/devices/")
	})
	if err != nil {
		return response, err
	}
	if response.StatusCode() == http.StatusOK {
		return response, nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return response, err
	}
	return response, errors.New("Error creating device")
}

// LoginDevice : Login Device
func LoginDevice(
	baseURL string,
	prn string,
	secret string,
) (
	string,
	error,
) {

	body := map[string]interface{}{}
	body["username"] = prn
	body["password"] = secret
	req := resty.R().SetBody(body)
	response, err := req.Post(baseURL + "/auth/login")
	if err != nil {
		return "", err
	}
	if response.StatusCode() == http.StatusOK {
		responseData := map[string]interface{}{}
		err = json.Unmarshal(response.Body(), &responseData)
		if err != nil {
			return "", err
		}
		return responseData["token"].(string), nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return "", err
	}
	return "", errors.New("Error login device")
}

// CreateTrail : Create Trail
func CreateTrail(baseURL string,
	deviceAccessToken string,
	state map[string]interface{},
) (
	*resty.Response,
	error,
) {
	req := resty.R().SetAuthToken(deviceAccessToken).SetBody(state)
	response, err := req.Post(baseURL + "/trails/")
	if err != nil {
		return response, err
	}
	if response.StatusCode() == http.StatusOK {
		return response, nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return response, err
	}
	return response, errors.New("Error creating trail")
}

// LogPrettyJSON : Pretty print Json content
func LogPrettyJSON(content []byte) error {
	var data interface{}
	err := json.Unmarshal(content, &data)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	fmt.Print("\n")
	return nil
}

// SetTempFilesInterrupHandler : Set Temp Files Interrup Handler
/*
This function will capture Interrupt signals and delete all temp files
*/
func SetTempFilesInterrupHandler(tempdir string) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(
		sigs,
		os.Interrupt,
	)
	go func() {
		<-sigs
		fileExist, err := IsFileExists(tempdir)
		if err != nil {
			log.Fatal(err.Error())
			os.Exit(1)
		}
		if fileExist {
			err := os.RemoveAll(tempdir)
			if err != nil {
				log.Fatal(err.Error())
				os.Exit(1)
			}
		}
		os.Exit(0)
	}()
}

// GetUserProfiles : Get User Profiles
func (s *Session) GetUserProfiles(baseURL string,
	userNick string,
) (
	*resty.Response,
	error,
) {
	response, err := s.DoAuthCall(func(req *resty.Request) (*resty.Response, error) {
		response, err := req.Get(baseURL + "/profiles/?nick=^" + userNick)
		return response, err
	})
	if err != nil {
		return response, err
	}
	if response.StatusCode() == http.StatusOK {
		return response, nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return response, err
	}
	return response, errors.New("Error getting user profile details")
}

// GetDevices : Get Devices
func (s *Session) GetDevices(baseURL string,
	ownerNick string,
	deviceNick string,
) (
	*resty.Response,
	error,
) {
	response, err := s.DoAuthCall(func(req *resty.Request) (*resty.Response, error) {
		ownerNickParam := ""
		if ownerNick != "" {
			ownerNickParam = "&owner-nick=" + ownerNick
		}
		return req.Get(baseURL + "/devices/?nick=^" + deviceNick + ownerNickParam)
	})

	if err != nil {
		return response, err
	}
	if response.StatusCode() == http.StatusOK {
		return response, nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return response, err
	}
	return response, errors.New("Error getting device details")
}

// GetDevice : Get Device
func (s *Session) GetDevice(baseURL string,
	deviceNick string,
	ownerNick string,
) (
	*resty.Response,
	error,
) {
	response, err := s.DoAuthCall(func(req *resty.Request) (*resty.Response, error) {

		if ownerNick != "" {
			req.SetQueryParam("owner-nick", ownerNick)
		}

		return req.Get(baseURL + "/devices/" + deviceNick)
	})
	if err != nil {
		return response, err
	}
	if response.StatusCode() == http.StatusOK {
		return response, nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return response, err
	}
	return response, errors.New("Error getting device details")
}

// SliceContainsItem : checks if an item exists in a string array or not
func SliceContainsItem(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func RemoveDirContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateDevice : Update  user-meta or device-meta field of a device
func (s *Session) UpdateDevice(
	baseURL string,
	deviceNick string,
	data map[string]interface{},
	updateField string,
) (
	*resty.Response,
	error,
) {
	response, err := s.DoAuthCall(func(req *resty.Request) (*resty.Response, error) {
		return req.SetBody(data).Patch(baseURL + "/devices/" + deviceNick + "/" + updateField)
	})
	if err != nil {
		return response, err
	}
	if response.StatusCode() == http.StatusOK {
		return response, nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return response, err
	}
	return response, errors.New("Error Updating " + updateField + " field")
}

// GetAuthStatus : Get Auth Status, GET /auth/auth_status
func (s *Session) GetAuthStatus(baseURL string) (
	*resty.Response,
	error,
) {
	response, err := s.DoAuthCall(func(req *resty.Request) (*resty.Response, error) {
		return req.Get(baseURL + "/auth/auth_status")
	})
	if err != nil {
		return response, err
	}
	if response.StatusCode() == http.StatusOK {
		return response, nil
	}
	//Logging error response
	err = LogPrettyJSON(response.Body())
	if err != nil {
		return response, err
	}
	return response, errors.New("Error getting auth status")
}

//ValidateSourceFlag : Validate Source Flag
func ValidateSourceFlag(source string) error {
	if source == "" {
		return nil
	}
	splits := strings.Split(source, ",")
	for _, v := range splits {
		if v != "remote" && v != "local" {
			return errors.New("Source flag only accepts remote / local, (e.g. --source=remote,local)")
		}
	}
	return nil
}

// ParseRFC3339 : Parse RFC3339 string : 2006-01-02T15:04:05+07:00
func ParseRFC3339(date string) (time.Time, error) {
	from, err := time.Parse("2006-01-02", date) //Date part only
	if err != nil {
		from, err = time.Parse("2006-01-02T15:04:05", date) //Date with time
		if err != nil {
			return time.Parse(time.RFC3339, date) //Date with time & timezone
		}
	}
	from = from.Local()
	return from, err
}

// SuggestNicks : Suggest Nicks (Either user nicks or device nicks)
func (s *Session) SuggestNicks(searchTerm string, baseURL string) {
	splits := strings.Split(searchTerm, "/")
	if len(splits) == 1 {
		searchTerm = splits[0]
		s.SuggestUserNicks(searchTerm, baseURL)
	} else if len(splits) == 2 {
		searchTerm = splits[1]
		s.SuggestDeviceNicks(splits[0], searchTerm, baseURL)
	}

}

// SuggestDeviceNicks : Suggest Device Nicks
func (s *Session) SuggestDeviceNicks(userNick, searchTerm string, baseURL string) {
	IsLoggedIn, err := s.IsUserLoggedIn(baseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}
	if !IsLoggedIn {
		fmt.Fprintln(os.Stderr, "Not-Loggedin -")
		return
	}

	devicesResponse, err := s.GetDevices(baseURL, userNick, searchTerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}
	responseData := []interface{}{}
	err = json.Unmarshal(devicesResponse.Body(), &responseData)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}
	if len(responseData) == 0 {
		fmt.Fprintln(os.Stderr, "No results")
	} else {
		for _, device := range responseData {
			if userNick != "" {
				fmt.Fprintln(os.Stderr, userNick+"/"+device.(map[string]interface{})["nick"].(string)+"\n")
			} else {
				fmt.Fprintln(os.Stderr, device.(map[string]interface{})["nick"].(string)+"\n")
			}

		}
	}
}

// SuggestUserNicks : Suggest User Nicks
func (s *Session) SuggestUserNicks(searchTerm string, baseURL string) {
	IsLoggedIn, err := s.IsUserLoggedIn(baseURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}
	if !IsLoggedIn {
		fmt.Fprintln(os.Stderr, "Not-Loggedin -")
		return
	}
	profilesResponse, err := s.GetUserProfiles(baseURL, searchTerm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}
	responseData := []interface{}{}
	err = json.Unmarshal(profilesResponse.Body(), &responseData)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error()+"\n")
		return
	}
	if len(responseData) == 0 {
		fmt.Fprintln(os.Stderr, "No results")
	} else {
		for _, profile := range responseData {
			if len(responseData) == 1 {
				fmt.Fprintln(os.Stderr, profile.(map[string]interface{})["nick"].(string)+"/\n")
			} else {
				fmt.Fprintln(os.Stderr, profile.(map[string]interface{})["nick"].(string)+"\n")
			}
		}
	}
}

// IsValidUrl tests a string to determine if it is a well-structured url or not.
func IsValidUrl(value string) bool {
	_, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}

	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

var tmplMap = map[string]interface{}{
	"basename": func(a string) string {
		return path.Base(a)
	},
	"sprintf": func(a, b string) string {
		return fmt.Sprintf(a, b)
	},
	"timeformat": func(a string, b time.Time) string {
		var r string
		switch a {
		case "ANSIC":
			r = b.Format(time.ANSIC)
		case "UnixDate":
			r = b.Format(time.UnixDate)
		case "RubyDate":
			r = b.Format(time.RubyDate)
		case "RFC822":
			r = b.Format(time.RFC822)
		case "RFC850":
			r = b.Format(time.RFC850)
		case "RFC1123":
			r = b.Format(time.RFC1123)
		case "RFC1123Z":
			r = b.Format(time.RFC1123Z)
		case "RFC3339":
			r = b.Format(time.RFC3339)
		case "RFC3339Nano":
			r = b.Format(time.RFC3339Nano)
		case "Kitchen":
			r = b.Format(time.Kitchen)
		case "Stamp":
			r = b.Format(time.Stamp)
		case "StampMilli":
			r = b.Format(time.StampMilli)
		case "StampMicro":
			r = b.Format(time.StampMicro)
		case "StampNano":
			r = b.Format(time.StampNano)
		default:
			r = b.Format(time.Stamp)
		}
		return r
	},
	"prn2id": func(a string) string {
		i := strings.LastIndex(a, "/")
		if i < 0 {
			return "INVALID_PRN(" + a + ")"
		}
		i += 1
		return a[i:]
	},
}

func SprintTmpl(format string, obj interface{}) (string, error) {
	var buf bytes.Buffer

	tmpl, err := template.New("template").
		Funcs(tmplMap).Funcs(sprig.GenericFuncMap()).
		Parse(format)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(&buf, obj)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
