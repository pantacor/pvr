/*
 * Copyright 2017  Alexander Sack <asac129@gmail.com>
 */
package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sethgrid/multibar"
	"github.com/urfave/cli"
	"gopkg.in/resty.v0"
)

var (
	hubBaseUrl string
)

//
// prn:<hostid>:<api>:version[?query1=one&query2=two]#[identifier]
type Prn struct {
	HostId    string
	Api       string
	Version   string
	Reference string
	Query     url.Values
	Raw       string
}

type AbstractCommand struct {
	Name       string
	AuthToken  string
	HubBaseUrl string
}

type CheckoutCommand struct {
	AbstractCommand
	DevicePrn Prn
	TargetDir string
	Revision  string
	Error     string
}

func NewPrn(devicePrn string) (Prn, error) {
	prn := Prn{}
	url, err := url.Parse(devicePrn)

	if err != nil {
		return Prn{}, err
	}
	prnparts := strings.Split(url.Opaque, ":")
	if len(prnparts) != 3 {
		return Prn{}, errors.New("Illegal Prn format. Allowed: prn:<hostid>:<api>:version[?query1=one&query2=two]#[identifier]")
	}

	prn.HostId = prnparts[0]
	prn.Api = prnparts[1]
	prn.Version = prnparts[2]
	prn.Reference = url.Fragment
	prn.Query = url.Query()
	prn.Raw = devicePrn

	return prn, nil
}

func DoCheckout(command CheckoutCommand) error {

	progressBars, _ := multibar.New()

	if command.DevicePrn.Api != "devices" {
		progressBars.Println(" - check device prn [ERROR]")
		return errors.New("Prn '" + command.DevicePrn.Raw + "' not a 'devices' prn")
	}
	_, err := os.Stat(command.TargetDir)
	if err == nil || !os.IsNotExist(err) {
		progressBars.Println(" - check target directory [ERROR]")
		return cli.NewExitError("Target directory ('"+command.TargetDir+"' already exists! See pvr checkout --help", 3)
	}

	err = os.MkdirAll(command.TargetDir, 0644)
	if err != nil {
		progressBars.Println(" - create target directory [ERROR]")
		return err
	}

	go progressBars.Listen()

	getUrl := command.HubBaseUrl + "/trails/" + command.DevicePrn.Reference
	resty.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	restyResponse, err := resty.R().SetAuthToken(command.AuthToken).Get(getUrl)
	if err != nil {
		progressBars.Println(" - get device info [ERROR]")
		return errors.New("ERROR: Cannot reach REST endpoint: '" + err.Error() + "'")
	}
	if restyResponse.StatusCode() != http.StatusOK {
		progressBars.Println(" - get device info [ERROR]")
		progressBars.Println("     STATUS: " + string(restyResponse.Status()))
		progressBars.Println("     BODY: " + string(restyResponse.Body()))
		return errors.New(restyResponse.Status() + "(" + getUrl + ")")
	}

	return nil
}

func PvrDirToJson(string) {

}

func main() {

	hubBaseUrl = "https://pantahub.appspot.com/api"

	app := cli.NewApp()
	app.Name = "pvr"
	app.Usage = "PantaVisor Remote"
	app.Version = "0.0.1"
	app.Action = func(c *cli.Context) error {
		fmt.Println("boom! I say!")
		return nil
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "access-token, a",
			Usage: "Use `ACCESS_TOKEN` for authorization with core services",
		},
		cli.StringFlag{
			Name:  "baseurl, b",
			Usage: "Use `BASEURL` for resolving prn URIs to core service endpoints",
		},
	}

	if os.Getenv("PANTAHUB_BASE") != "" {
		hubBaseUrl = os.Getenv("PANTAHUB_BASE")
	}

	app.Commands = []cli.Command{
		CommandInit(),
		CommandAdd(),
		CommandJson(),
	}
	app.Run(os.Args)
}
