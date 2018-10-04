package libpvr

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/go-resty/resty"
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

	accountsEp := u1.String() + "/accounts"

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
