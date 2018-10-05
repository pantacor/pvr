package libpvr

import (
	"errors"
	"os/user"
	"path/filepath"

	"github.com/go-resty/resty"
	"github.com/urfave/cli"
)

func GetConfigDir() (string, error) {

	user, err := user.Current()
	if err != nil {
		return "", err
	}

	return filepath.Join(user.HomeDir, ".pvr"), nil
}

type Session struct {
	app  *cli.App
	auth *PvrAuthConfig
}

func NewSession(app *cli.App) (*Session, error) {
	configDir, err := GetConfigDir()

	if err != nil {
		return nil, errors.New("Cannot get config dir: " + err.Error())
	}

	configPath := filepath.Join(configDir, "auth.json")
	authConfig, err := LoadConfig(configPath)

	if err != nil {
		return nil, errors.New("Cannot load config: " + err.Error())
	}

	return &Session{
		app:  app,
		auth: authConfig,
	}, nil
}

func (s *Session) GetApp() *cli.App {
	return s.app
}

func (s *Session) DoAuthCall(fn WrappableRestyCallFunc) (*resty.Response, error) {

	var bearer string
	var err error
	var response *resty.Response

	// legacy flat -a from CLI will give a default token
	bearer = s.GetApp().Metadata["PVR_AUTH"].(string)
	response, err = fn(resty.R().SetAuthToken(bearer))

	// if we see www-authenticate, we need to auth ...
	authHeader := response.Header().Get("www-authenticate")

	// first try cached accesstoken
	if authHeader != "" {
		bearer, err = s.auth.getCachedAccessToken(authHeader)
		if bearer != "" {
			response, err = fn(resty.R().SetAuthToken(bearer))
			authHeader = response.Header().Get("Www-Authenticate")
		}
	}

	// then get new accesstoken
	if authHeader != "" {
		bearer, err = s.auth.getNewAccessToken(authHeader)
		if bearer != "" {
			response, err = fn(resty.R().SetAuthToken(bearer))
			authHeader = response.Header().Get("Www-Authenticate")
		}
	}

	return response, err
}
