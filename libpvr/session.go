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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-resty/resty"
	"github.com/urfave/cli"
)

const (
	ConfigurationFile = "config.json"
)

type Session struct {
	app           *cli.App
	auth          *PvrAuthConfig
	Configuration *PvrGlobalConfig
	configDir     string
}

func NewSession(app *cli.App) (*Session, error) {
	configDir := app.Metadata["PVR_CONFIG_DIR"].(string)
	authConfigPath := filepath.Join(configDir, "auth.json")
	generalConfigPath := filepath.Join(configDir, ConfigurationFile)

	authConfig, err := LoadConfig(authConfigPath)
	if err != nil {
		return nil, errors.New("Cannot load config: " + err.Error())
	}

	configuration, err := LoadConfiguration(generalConfigPath)
	if err != nil {
		return nil, err
	}

	return &Session{
		app:           app,
		auth:          authConfig,
		configDir:     configDir,
		Configuration: configuration,
	}, nil
}

func (s *Session) GetConfigDir() string {
	return s.configDir
}

func (s *Session) GetApp() *cli.App {
	return s.app
}

// Login : Login to a given URL
func (s *Session) Login(APIURL string, withAnon bool) (*resty.Response, error) {
	//clear token
	u, err := url.Parse(APIURL)
	if err != nil {
		return nil, err
	}
	host := u.Scheme + "://" + u.Host
	authHeader := "JWT realm=\"pantahub services\", ph-aeps=\"" + host + "/auth\""
	err = s.auth.resetCachedAccessToken(authHeader)
	if err != nil {
		return nil, err
	}
	response, err := s.DoAuthCall(withAnon, func(req *resty.Request) (*resty.Response, error) {
		return req.Get(APIURL)
	})
	if err != nil {
		return response, err
	}
	return response, nil
}

func (s *Session) DoAuthCall(withAnon bool, fn WrappableRestyCallFunc) (*resty.Response, error) {

	var bearer string
	var err error
	var response *resty.Response
	var authHeader string
	var anonTried bool
	var interactive bool

	bearer = s.GetApp().Metadata["PVR_AUTH"].(string)
	if bearer == "" {
		interactive = true
	}

	for {
		var newAuthHeader string
		// legacy flat -a from CLI will give a default token
		response, err = fn(resty.R().SetAuthToken(bearer))
		// we continue looping for 401 and 403 error codes, everything
		// else are error conditions that need to be handled upstream
		if err == nil &&
			response.StatusCode() != http.StatusUnauthorized &&
			response.StatusCode() != http.StatusForbidden {
			break
		}
		if !interactive {
			break
		}
		// if we see www-authenticate, we need to auth ...
		newAuthHeader = response.Header().Get("www-authenticate")

		// first try cached accesstoken or get a new one
		if newAuthHeader != "" {
			// if we already had one run with this auth header, evict from cache
			if authHeader != "" {
				s.auth.resetCachedAccessToken(authHeader)
			}
			authHeader = newAuthHeader

			bearer, err = s.auth.getCachedAccessToken(authHeader)
			if err != nil {
				break
			}
			if bearer == "" {
				bearer, err = s.auth.getNewAccessToken(authHeader, true, withAnon)
				anonTried = withAnon
			}
			if err != nil {
				break
			}

		} else if response.StatusCode() == http.StatusForbidden {
			if anonTried {
				fmt.Fprintln(os.Stderr, "** ACCESS DENIED: user cannot access repository. **")
			}
			bearer, err = s.auth.getNewAccessToken(authHeader, false, withAnon && !anonTried)
			anonTried = withAnon
			if err != nil {
				break
			}
		}

		// now that we would have a refreshed bearer, lets go again
		s.GetApp().Metadata["PVR_AUTH"] = bearer
	}

	return response, err
}
