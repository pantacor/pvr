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
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
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
	"gitlab.com/pantacor/pvr/utils/pvjson"
)

type PvsMatch struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type PvsOptions struct {
	Algorithm      gojose.SignatureAlgorithm
	X5cPath        string
	ExtraHeaders   map[string]interface{}
	IncludePayLoad bool
	OutputFile     *os.File
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

	err := pvjson.Unmarshal(buf, &bufMap)
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

	err := pvjson.Unmarshal(buf, &m)

	if err != nil {
		return nil, err
	}

	delete(m, "payload")

	return cjson.Marshal(m)
}

func parseCertsFromPEMFile(path string) ([]*x509.Certificate, error) {

	var certs []*x509.Certificate

	pemCerts, err := os.ReadFile(path)

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

	fileBuf, err := os.ReadFile(pvsPath)
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
	err = pvjson.Unmarshal(jsonBuf, &match)
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

	fileBuf, err := io.ReadAll(f)
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
	if parsedKey, err = x509.ParseECPrivateKey(privPemBytes); err == nil { // note this returns type `interface{}`
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

	strippedBuf := []byte(sig.FullSerialize())
	if !options.IncludePayLoad {
		strippedBuf, err = stripPayloadFromRawJSON(strippedBuf)
	}

	if err != nil {
		return err
	}

	sigMap := map[string]interface{}{}

	err = pvjson.Unmarshal(strippedBuf, &sigMap)

	if err != nil {
		return err
	}

	sigMap["#spec"] = "pvs@2"

	newJson, err := cjson.Marshal(sigMap)
	if err != nil {
		return err
	}

	if options.OutputFile != nil {
		_, err = options.OutputFile.Write(newJson)

		if err != nil {
			return err
		}
	}
	err = os.MkdirAll(path.Join(p.Dir, "_sigs"), 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(path.Join(p.Dir, "_sigs", name+".json"), newJson, 0644)
	if err != nil {
		return err
	}

	return nil
}

type PvsCertPool struct {
	certPool *x509.CertPool
	certsRaw [][]byte
}

func NewPvsCertPool() *PvsCertPool {
	return &PvsCertPool{
		certPool: x509.NewCertPool(),
	}
}

func (s *PvsCertPool) GetCertsRaw() [][]byte {
	return s.certsRaw
}

func (s *PvsCertPool) AppendCertsFromPEM(pemCerts []byte) (ok bool) {
	ok = s.certPool.AppendCertsFromPEM(pemCerts)
	if !ok {
		return ok
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
		_, err := x509.ParseCertificate(certBytes)
		if err != nil {
			continue
		}
		s.certsRaw = append(s.certsRaw, certBytes)
		ok = true
	}

	return ok
}

type JwsVerifySummary struct {
	Protected       []string      `json:"protected,omitempty"`
	Excluded        []string      `json:"excluded,omitempty"`
	NotSeen         []string      `json:"notseen,omitempty"`
	FullJSONWebSigs []interface{} `json:"sigs,omitempty"`
}

// JwsVerify will add or update a signature based using a private
// key provided.
//
// The payload will be assembled from the prinstine system state JSON
// using the match rule provided in PvsMatch struct included in the pvs.
//
// special value for caCerts "_system_" hints at using the system cacert
// store. Can be configured using SSH_CERT_FILE and SSH_CERTS_DIR on linux
func (p *Pvr) JwsVerifyPvs(keyPath string, caCerts string, pvsPath string, includePayload bool) (*JwsVerifySummary, error) {

	var summary JwsVerifySummary
	var err error

	buf := p.PristineJson

	if buf == nil {
		return nil, errors.New("empty state format")
	}

	var pubKeys []interface{}
	var certPool *PvsCertPool

	if keyPath != "" {
		f, err := os.Open(keyPath)

		if err != nil {
			return nil, err
		}

		fileBuf, err := io.ReadAll(f)
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
			var pubKey interface{}
			switch v := parsedKey.(type) {
			case *rsa.PublicKey:
				pubKey = v
			case *ecdsa.PublicKey:
				pubKey = v
			default:
				fmt.Fprintf(os.Stderr, "WARNING: casting pubKey key of type "+reflect.TypeOf(parsedKey).String())
				continue
			}

			pubKeys = append(pubKeys, pubKey)
			if IsDebugEnabled {
				fmt.Fprintf(os.Stderr, "INFO: added pubkey: %d\n", len(pubKeys))
			}
		}
	} else if caCerts == "_system_" {
		return nil, fmt.Errorf("Using system cert pool not supported anymore")
	} else if caCerts != "" {
		certPool = NewPvsCertPool()
		if err != nil {
			return nil, err
		}
		caCertsBuf, err := os.ReadFile(caCerts)
		if err != nil {
			return nil, err
		}

		ok := certPool.AppendCertsFromPEM(caCertsBuf)
		if !ok {
			fmt.Fprintln(os.Stderr, "WARNING: could not append cacerts to cert pool. Disabling cert pool.")
			certPool = nil
		}
	}

	fileBuf, err := os.ReadFile(pvsPath)
	if err != nil {
		return nil, err
	}

	var previewSig map[string]interface{}
	err = pvjson.Unmarshal(fileBuf, &previewSig)

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
	err = pvjson.Unmarshal(jsonBuf, &match)
	if err != nil {
		return nil, err
	}

	selection, err := selectPayload(buf, match)

	if err != nil {
		return nil, err
	}

	payloadBuf, err := selectPayloadBuf(buf, match)

	var verified bool
	var pemcerts [][]*x509.Certificate

	if certPool != nil {
		ku := []x509.ExtKeyUsage{
			x509.ExtKeyUsageCodeSigning,
		}

		pemcerts, err = sig.Signatures[0].Header.
			Certificates(x509.VerifyOptions{
				Roots:     certPool.certPool,
				KeyUsages: ku,
			})

		// if we had x5c and chain could be validated we use just that cert
		if err == nil {
			pubKeys = append(pubKeys, pemcerts[0][0].PublicKey)
		} else if err != nil && err.Error() == "go-jose/go-jose: no x5c header present in message" {
			// we manuall iterate the system pool if x5c is not there....
			err = nil
			for _, derCandidate := range certPool.certsRaw {
				cert, err := x509.ParseCertificate(derCandidate)
				if err != nil {
					fmt.Fprintf(os.Stderr, "a cert in the pool cannot be parsed %s\n", err.Error())
					continue
				}
				pemcerts, err = cert.Verify(x509.VerifyOptions{
					Roots:     certPool.certPool,
					KeyUsages: ku,
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "could not validate cert in pool against the pool itself %s\n", err.Error())
					continue
				}
				pubKeys = append(pubKeys, pemcerts[0][0].PublicKey)
			}
		} else {
			// we just continue as we allow to still validate through manually set pubKeys below...
		}
	}

	for _, pubKey := range pubKeys {
		switch pubKey.(type) {
		case *rsa.PublicKey:
		case *ecdsa.PublicKey:
		default:
			fmt.Fprintf(os.Stderr, "WARNING: validation pub key not of supported type: '%s'\n", reflect.TypeOf(pubKey).String())
			continue
		}

		if IsDebugEnabled {
			fmt.Fprintf(os.Stderr, "Validating payload: '%s'\n", string(payloadBuf))
		}
		err = sig.DetachedVerify(payloadBuf, pubKey)
		if err == nil {
			verified = true
			break
		}
	}

	if len(pubKeys) == 0 {
		return nil, fmt.Errorf("no pubKeys available. Neither as parameters nor from x5c header nor from root pool")
	}

	if !verified {
		return nil, fmt.Errorf("could not verify payload %s (error= %w)", string(payloadBuf), err)
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

	var fullSig map[string]interface{}

	err = pvjson.Unmarshal([]byte(sig.FullSerialize()), &fullSig)

	if err != nil {
		return nil, err
	}

	if includePayload {
		buf := make([]byte, base64.StdEncoding.
			WithPadding(base64.NoPadding).EncodedLen(len(payloadBuf)))
		base64.StdEncoding.
			WithPadding(base64.NoPadding).Encode(buf, payloadBuf)
		fullSig["payload"] = string(buf)
	}
	summary.FullJSONWebSigs = append(summary.FullJSONWebSigs, fullSig)

	return &summary, nil
}
