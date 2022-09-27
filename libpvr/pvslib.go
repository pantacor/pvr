//
// Copyright 2021  Pantacor Ltd.
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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/bmatcuk/doublestar"
	cjson "github.com/gibson042/canonicaljson-go"
	gojose "github.com/go-jose/go-jose/v3"
	"github.com/go-resty/resty"
)

type PvsMatch struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type PvsOptions struct {
	Algorithm    gojose.SignatureAlgorithm
	X5cPath      string
	ExtraHeaders map[string]interface{}
}

type PvsPartSelection struct {
	Selected    map[string]interface{}
	NotSelected map[string]interface{}
	NotSeen     map[string]interface{}
}

const (
	tarFileName       = "pvs.defaultkeys.tar.gz"
	SigKeyFilename    = "key.default.pem"
	SigX5cFilename    = "x5c.default.pem"
	SigCacertFilename = "cacerts.default.pem"
)

func GetFromConfigPvs(url, configPath, name string) (string, error) {
	cert := path.Join(configPath, "pvs", name)
	if _, err := os.Stat(cert); errors.Is(err, os.ErrNotExist) {
		err := DownloadSigningCertWithConfirmation(url, configPath)
		if err != nil {
			return "", err
		}
	}

	return cert, nil
}

// DownloadSigningCertWithConfirmation ask for confirmation to download signing certs
func DownloadSigningCertWithConfirmation(url, path string) error {
	question := fmt.Sprintf(
		"%s %s\n%s",
		"Do you want to download the default",
		"developer keys that can be validated by pantavisor official developer builds? [yes/no]",
		"(NOTE: this should be used only for development proposes because they are not secret)",
	)
	c := AskForConfirmation(question)
	if !c {
		return nil
	}

	fmt.Printf("Downloading certs from: \n%s\n", url)
	return DownloadSigningCert(url, path)
}

func DownloadSigningCert(url, path string) error {
	tarFilePath := filepath.Join(path, tarFileName)
	resp, err := resty.
		R().
		SetOutput(tarFilePath).
		Get(url)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return errors.New("there was an error downloading the certificates tar.gz")
	}

	err = Untar(path, tarFilePath, nil)
	if err != nil {
		return err
	}

	return nil
}

func selectPayload(buf []byte, match *PvsMatch) (*PvsPartSelection, error) {
	var bufMap map[string]interface{}
	selection := PvsPartSelection{}

	selection.Selected = map[string]interface{}{}
	selection.NotSelected = map[string]interface{}{}
	selection.NotSeen = map[string]interface{}{}

	err := json.Unmarshal(buf, &bufMap)
	if err != nil {
		return nil, err
	}

	for k, v := range bufMap {
		var found interface{}
		key := ""
		var excludedMatch bool

		for _, I := range match.Include {
			m, err := doublestar.Match(I, k)
			if err != nil {
				return nil, err
			}
			if !m {
				continue
			}

			found = v
			key = k
			break
		}

		excludedMatch = false
		for _, E := range match.Exclude {
			m, err := doublestar.Match(E, k)
			if err != nil {
				return nil, err
			}
			if !m {
				continue
			}
			found = nil
			key = ""
			excludedMatch = true
			break
		}

		if key != "" {
			selection.Selected[key] = found
		} else if excludedMatch {
			selection.NotSelected[k] = v
		} else {
			selection.NotSeen[k] = v
		}
	}

	return &selection, nil
}

func selectPayloadBuf(buf []byte, match *PvsMatch) ([]byte, error) {

	selection, err := selectPayload(buf, match)

	if err != nil {
		return nil, err
	}

	resBuf, err := cjson.Marshal(selection.Selected)

	if err != nil {
		return nil, err
	}

	return resBuf, nil
}

func stripPayloadFromRawJSON(buf []byte) ([]byte, error) {

	var m map[string]interface{}

	err := json.Unmarshal(buf, &m)

	if err != nil {
		return nil, err
	}

	delete(m, "payload")

	return cjson.Marshal(m)
}

func parseCertsFromPEMFile(path string) ([]*x509.Certificate, error) {

	var certs []*x509.Certificate

	pemCerts, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	for len(pemCerts) > 0 {
		var block *pem.Block
		block, pemCerts = pem.Decode(pemCerts)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
			continue
		}

		certBytes := block.Bytes
		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			continue
		}

		certs = append(certs, cert)
	}

	return certs, nil
}

func (p *Pvr) Verify(keyPath string) error {

	return errors.New("NOT IMPLEMENTED")
}
func (p *Pvr) JwsSignAuto(keyPath string, part string, options *PvsOptions) error {

	return errors.New("NOT IMPLEMENTED")
}

// JwsSignPvs will parse a pvs@s json provided as argument and
// use the included PvsMatch section to invoke JwsSign
func (p *Pvr) JwsSignPvs(privKeyPath string,
	pvsPath string,
	options *PvsOptions) error {

	buf := p.PristineJson
	if buf == nil {
		return errors.New("empty state format")
	}

	fileBuf, err := ioutil.ReadFile(pvsPath)
	if err != nil {
		return err
	}

	sig, err := gojose.ParseSigned(string(fileBuf))
	if err != nil {
		return err
	}

	header := sig.Signatures[0].Protected
	pvsHeader := header.ExtraHeaders[gojose.HeaderKey("pvs")]
	jsonBuf, err := json.Marshal(pvsHeader)
	if err != nil {
		return err
	}

	var match *PvsMatch
	err = json.Unmarshal(jsonBuf, &match)
	if err != nil {
		return err
	}

	name := path.Base(pvsPath)
	name = strings.TrimSuffix(name, ".json")
	return p.JwsSign(name, privKeyPath, match, options)
}

// JwsSign will add or update a signature based using a private
// key provided.
//
// The payload will be assembled from the prinstine system state JSON
// using the match rule provided in PvsMatch struct.
//
// PvsOptions allow to pass additional JoseHeader options to include
// in the Signature.
func (p *Pvr) JwsSign(name string,
	privKeyPath string,
	match *PvsMatch,
	options *PvsOptions) error {

	var signKey *pem.Block

	buf := p.PristineJson

	if buf == nil {
		return errors.New("empty state format")
	}

	f, err := os.Open(privKeyPath)

	if err != nil {
		return err
	}

	fileBuf, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	rest := fileBuf
	for {
		var p *pem.Block
		p, rest = pem.Decode(rest)
		if p == nil {
			break
		}
		if strings.Contains(p.Type, "PRIVATE KEY") {
			signKey = p
			break
		}
	}
	if signKey == nil {
		return errors.New("No valid PEM encoded sign key found in " + privKeyPath)
	}

	var algo gojose.SignatureAlgorithm
	var parsedKey interface{}
	privPemBytes := signKey.Bytes
	if parsedKey, err = x509.ParsePKCS1PrivateKey(privPemBytes); err == nil {
		goto found
	}
	if parsedKey, err = x509.ParsePKCS8PrivateKey(privPemBytes); err == nil { // note this returns type `interface{}`
		goto found
	}

	return errors.New("Private key cannot be parsed." + err.Error())

found:
	var ok bool
	var privKey interface{}
	if privKey, ok = parsedKey.(*rsa.PrivateKey); ok {
		ok = true
		algo = gojose.RS256
	} else if privKey, ok = parsedKey.(*ecdsa.PrivateKey); ok {
		ok = true
		ePK := privKey.(*ecdsa.PrivateKey)
		switch ePK.Params().BitSize {
		case 256:
			algo = gojose.ES256
		case 384:
			algo = gojose.ES384
		case 512:
			algo = gojose.ES512
		default:
			return errors.New("Private key with unsupported bitsize for ESXXXX: " + fmt.Sprint(ePK.Params().BitSize))
		}
	} else {
		return errors.New("Not supported priv key of type " + reflect.TypeOf(parsedKey).Name())
	}

	if match.Include == nil {
		match.Include = []string{
			"*",
		}
	}

	if match.Exclude == nil {
		match.Exclude = []string{
			path.Join("**", "src.json"),
			path.Join("_sigs", name+".json"),
		}
	}

	signerOpts := &gojose.SignerOptions{EmbedJWK: true}
	signerOpts = signerOpts.
		WithHeader("pvs", match).
		WithType("PVS")

	if options.X5cPath != "" {
		certs, err := parseCertsFromPEMFile(options.X5cPath)
		if err != nil {
			return err
		}

		var certsRaw [][]byte

		for _, c := range certs {
			certsRaw = append(certsRaw, c.Raw)
		}

		signerOpts.WithHeader("x5c", certsRaw)
	}

	for k, v := range options.ExtraHeaders {
		signerOpts = signerOpts.WithHeader(gojose.HeaderKey(k), v)
	}

	signer, err := gojose.NewSigner(gojose.SigningKey{
		Algorithm: algo,
		Key:       privKey,
	}, signerOpts)

	if err != nil {
		return err
	}

	payloadBuf, err := selectPayloadBuf(buf, match)

	if err != nil {
		return err
	}

	sig, err := signer.Sign(payloadBuf)
	if err != nil {
		return err
	}

	strippedBuf, err := stripPayloadFromRawJSON([]byte(sig.FullSerialize()))

	if err != nil {
		return err
	}

	sigMap := map[string]interface{}{}

	err = json.Unmarshal(strippedBuf, &sigMap)

	if err != nil {
		return err
	}

	sigMap["#spec"] = "pvs@2"

	newJson, err := cjson.Marshal(sigMap)
	if err != nil {
		return err
	}

	err = os.MkdirAll(path.Join(p.Dir, "_sigs"), 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path.Join(p.Dir, "_sigs", name+".json"), newJson, 0644)
	if err != nil {
		return err
	}

	return nil
}

type JwsVerifySummary struct {
	Protected []string `json:"protected,omitempty"`
	Excluded  []string `json:"excluded,omitempty"`
	NotSeen   []string `json:"notseen,omitempty"`
}

// JwsVerify will add or update a signature based using a private
// key provided.
//
// The payload will be assembled from the prinstine system state JSON
// using the match rule provided in PvsMatch struct included in the pvs.
//
// special value for caCerts "_system_" hints at using the system cacert
// store. Can be configured using SSH_CERT_FILE and SSH_CERTS_DIR on linux
//
func (p *Pvr) JwsVerifyPvs(keyPath string, caCerts string, pvsPath string) (*JwsVerifySummary, error) {

	var summary JwsVerifySummary
	var err error

	buf := p.PristineJson

	if buf == nil {
		return nil, errors.New("empty state format")
	}

	var pubKeys []interface{}
	var certPool *x509.CertPool

	if keyPath != "" {
		f, err := os.Open(keyPath)

		if err != nil {
			return nil, err
		}

		fileBuf, err := ioutil.ReadAll(f)
		if err != nil {
			return nil, err
		}
		rest := fileBuf
		var pubPems []*pem.Block

		for rest != nil {
			var p *pem.Block
			p, rest = pem.Decode(rest)
			if p == nil {
				break
			}
			if strings.HasPrefix(p.Type, "PUBLIC ") {
				pubPems = append(pubPems, p)
			}
		}

		if pubPems == nil {
			return nil, errors.New("No valid PEM encoded verify key found " + keyPath)
		}

		for _, pubPem := range pubPems {
			var parsedKey interface{}
			pemBytes := pubPem.Bytes
			if parsedKey, err = x509.ParsePKCS1PublicKey(pemBytes); err != nil {
				if parsedKey, err = x509.ParsePKIXPublicKey(pemBytes); err != nil {
					fmt.Fprintf(os.Stderr, "WARNING: Error Parsing Public key from PEM: %s\n", err.Error())
					continue
				}
			}
			var ok bool
			var pubKey interface{}
			pubKey, ok = parsedKey.(*rsa.PublicKey)

			if pubKey == nil {
				pubKey, ok = parsedKey.(*ecdsa.PublicKey)
			}
			if !ok {
				fmt.Fprintf(os.Stderr, "WARNING: casting pubey key\n")
				continue
			}
			pubKeys = append(pubKeys, pubKey)
			if IsDebugEnabled {
				fmt.Fprintf(os.Stderr, "INFO: added pubey: %d\n", len(pubKeys))
			}
		}
	} else if caCerts == "_system_" {
		certPool, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(os.Stderr, "Using system cert pool")
	} else if caCerts != "" {
		certPool = x509.NewCertPool()
		if err != nil {
			return nil, err
		}
		caCertsBuf, err := ioutil.ReadFile(caCerts)
		if err != nil {
			return nil, err
		}

		ok := certPool.AppendCertsFromPEM(caCertsBuf)
		if !ok {
			fmt.Fprintln(os.Stderr, "WARNING: could not append cacerts to cert pool. Disabling cert pool.")
			certPool = nil
		}
	}

	fileBuf, err := ioutil.ReadFile(pvsPath)
	if err != nil {
		return nil, err
	}

	var previewSig map[string]interface{}
	err = json.Unmarshal(fileBuf, &previewSig)

	if err != nil {
		return nil, err
	}

	if _, ok := previewSig["#spec"]; !ok {
		return nil, errors.New("ERROR: pvs signature does not have a #spec element")
	}

	if previewSig["#spec"].(string) != "pvs@2" {
		return nil, errors.New("ERROR: pvs signature must be of #spec pvs@2")
	}

	sig, err := gojose.ParseSigned(string(fileBuf))

	if err != nil {
		return nil, err
	}

	header := sig.Signatures[0].Protected
	pvsHeader := header.ExtraHeaders[gojose.HeaderKey("pvs")]

	jsonBuf, err := json.Marshal(pvsHeader)
	if err != nil {
		return nil, err
	}

	var match *PvsMatch
	err = json.Unmarshal(jsonBuf, &match)
	if err != nil {
		return nil, err
	}

	selection, err := selectPayload(buf, match)

	if err != nil {
		return nil, err
	}

	payloadBuf, err := selectPayloadBuf(buf, match)

	var verified bool

	for _, pk := range pubKeys {

		err = sig.DetachedVerify(payloadBuf, pk)
		if err != nil {
			continue
		}
	}

	var pemcerts [][]*x509.Certificate

	if !verified && certPool != nil {
		ku := []x509.ExtKeyUsage{
			x509.ExtKeyUsageCodeSigning,
		}

		pemcerts, err = sig.Signatures[0].Header.
			Certificates(x509.VerifyOptions{
				Roots:     certPool,
				KeyUsages: ku,
			})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting up and validating certificates %s\n", err.Error())
		} else {
			var pubKey interface{}
			var ok bool
			pubKey, ok = pemcerts[0][0].PublicKey.(*rsa.PublicKey)
			if !ok || pubKey == nil {
				pubKey, ok = pemcerts[0][0].PublicKey.(*ecdsa.PublicKey)
			}
			if ok {
				if IsDebugEnabled {
					fmt.Fprintf(os.Stderr, "Validating payload: '%s'\n", string(payloadBuf))
				}
				err = sig.DetachedVerify(payloadBuf, pubKey)
			} else {
				err = errors.New("error retrieving public from certs key")
			}
		}

		if err == nil {
			verified = true
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR verifying: "+err.Error())
	}

	if !verified && err != nil && pubKeys != nil {
		for _, pubKey := range pubKeys {
			if err = sig.DetachedVerify(payloadBuf, pubKey); err == nil {
				verified = true
				fmt.Fprintf(os.Stderr, "ERROR verifying payload: "+err.Error())
				break
			}
		}
	}

	if !verified {
		return nil, errors.New("error validating signature from system cert pool and from provided pubKey file.")
	}

	for k := range selection.Selected {
		summary.Protected = append(summary.Protected, k)
	}
	for k := range selection.NotSelected {
		summary.Excluded = append(summary.Excluded, k)
	}
	for k := range selection.NotSeen {
		summary.NotSeen = append(summary.NotSeen, k)
	}

	return &summary, nil
}