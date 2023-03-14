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
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	pvrapi "gitlab.com/pantacor/pvr/api"
	"gitlab.com/pantacor/pvr/utils/pvjson"
)

// RemoteCopy will perform a remote only copy
// by taking the json, select elements that have the #fragement of
// a provided url as Prefix of their key and replace all elements
// in pvrDest that match that prefix.
// The target prefix can be overloaded through providing a #fragment
// in destination URL as well.
// It is an illegal input if pvrSrc has no fragement, but pvrDest has one.
// It is however legal input if pvrSrc has a fragement, but pvrDest does not
// have one. In that case the same fragement is implicitely appended to pvrDest
func (p *Pvr) RemoteCopy(pvrSrc string, pvrDest string, merge bool,
	envelope string, commitMsg string, rev int, force bool) error {

	srcURL, err := url.Parse(pvrSrc)
	if err != nil {
		return err
	}
	if !srcURL.IsAbs() {
		repoURL := p.Session.GetApp().Metadata["PVR_REPO_BASEURL_url"].(*url.URL)
		srcURL = repoURL.ResolveReference(srcURL)
	}

	destURL, err := url.Parse(pvrDest)
	if err != nil {
		return err
	}
	if !destURL.IsAbs() {
		repoURL := p.Session.GetApp().Metadata["PVR_REPO_BASEURL_url"].(*url.URL)
		destURL = repoURL.ResolveReference(destURL)
	}

	srcRemote, err := p.initializeRemote(srcURL)

	if err != nil {
		return err
	}

	destRemote, err := p.initializeRemote(destURL)

	if err != nil {
		return err
	}

	srcJsonBuf, err := p.getJSONBuf(srcRemote)
	if err != nil {
		return err
	}

	destJsonBuf, err := p.getJSONBuf(destRemote)
	if err != nil {
		return err
	}

	_, dest, err := PatchState(srcJsonBuf, destJsonBuf, srcURL.Fragment, destURL.Fragment, merge, nil)
	if err != nil {
		return err
	}

	buf, err := p.postRemoteJson(destRemote, dest, envelope, commitMsg, rev, force)
	if err != nil {
		return err
	}

	responseMap := map[string]interface{}{}

	err = pvjson.Unmarshal(buf, &responseMap)

	if err != nil {
		return err
	}

	revNumber := responseMap["rev"].(json.Number)
	fmt.Fprintf(os.Stderr, "Successfully posted Revision %s (%s) to device id %s\n", revNumber.String(),
		responseMap["state-sha"].(string)[:8], responseMap["trail-id"])

	return nil
}

func (p *Pvr) RemoteInfo(pvrRef string) (*pvrapi.PvrRemote, error) {
	infoUrl, err := url.Parse(pvrRef)
	if err != nil {
		return nil, err
	}
	if !infoUrl.IsAbs() {
		repoURL := p.Session.GetApp().Metadata["PVR_REPO_BASEURL_url"].(*url.URL)
		infoUrl = repoURL.ResolveReference(infoUrl)
	}

	pvrRemote, err := p.initializeRemote(infoUrl)

	if err != nil {
		return nil, err
	}

	return &pvrRemote, nil
}
