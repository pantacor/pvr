//
// Copyright 2018  Pantacor Ltd.
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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/go-resty/resty"
)

const (
	// PhTrailsEp constant defining pantahub /trails
	PhTrailsEp = "/trails"

	// PhTrailsSummaryEp constant defines pantahub /trails/summary EP
	PhTrailsSummaryEp = PhTrailsEp + "/summary"

	// PhAccountsEp constant defines pantahub /accounts endpoint
	PhAccountsEp = "/auth/accounts"
)

func DoRegister(authEp, email, username, password string) error {

	if authEp == "" {
		return errors.New("DoRegister: no authentication endpoint provided.")
	}
	if email == "" {
		return errors.New("DoRegister: no email provided.")
	}
	if username == "" {
		return errors.New("DoRegister: no username provided.")
	}
	if password == "" {
		return errors.New("DoRegister: no password provided.")
	}

	u1, err := url.Parse(authEp)
	if err != nil {
		return errors.New("DoRegister: error parsing EP url.")
	}

	accountsEp := u1.String() + PhAccountsEp

	m := map[string]string{
		"email":    email,
		"nick":     username,
		"password": password,
	}

	response, err := resty.R().SetBody(m).
		Post(accountsEp)

	if err != nil {
		log.Fatal("Error calling POST for registration: " + err.Error())
		return err
	}

	m1 := map[string]interface{}{}
	err = json.Unmarshal(response.Body(), &m1)

	if err != nil {
		log.Fatal("Error parsing Register body(" + err.Error() + ") for " + accountsEp + ": " + string(response.Body()))
		return err
	}

	if response.StatusCode() != 200 {
		return errors.New("Failed to register: " + string(response.Body()))
	}

	fmt.Println("Registration Response: " + string(response.Body()))

	return nil
}

type PantahubDevice struct {
	Id               string    `json:"deviceid"`
	Prn              string    `json:"device"`
	Nick             string    `json:"device-nick"`
	Revision         int       `json:"revision"`
	ProgressRevision int       `json:"progress-revision"`
	Timestamp        time.Time `json:"timestamp"`
	StateSha         string    `json:"state-sha"`
	Status           string    `json:"status"`
	StatusMsg        string    `json:"status-msg"`
}

func (p *Session) DoPs(baseurl string) ([]PantahubDevice, error) {
	res, err := p.DoAuthCall(func(req *resty.Request) (*resty.Response, error) {
		burl, err := url.Parse(baseurl)
		if err != nil {
			return nil, errors.New("Cannot parse baseurl '" + baseurl + "': " + err.Error())
		}

		trailSummaryEpURL, err := url.Parse(PhTrailsSummaryEp)
		if err != nil {
			return nil, errors.New("Cannot parse trailsSummaryEpURL '" + trailSummaryEpURL.String() + "': " + err.Error())
		}

		fullURL := burl.ResolveReference(trailSummaryEpURL)
		return req.Get(fullURL.String())
	})

	if err != nil {
		return nil, errors.New("ERROR: authenticated call to " + baseurl + " failed with: " + err.Error())
	}

	var resultSet []PantahubDevice
	err = json.Unmarshal(res.Body(), &resultSet)

	if err != nil {
		return nil, errors.New("ERROR: cannot decode result of authenticated call to " + baseurl + ": " + err.Error())
	}

	return resultSet, nil
}
