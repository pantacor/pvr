//
// Copyright 2020  Pantacor Ltd.
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
	"net/url"
	"os"
	"strings"
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

	srcFrag := srcURL.Fragment
	destFrag := destURL.Fragment

	if srcFrag == "" && destFrag != "" {
		return errors.New("RemoteCopy source URL must have a #fragement part if destination URL is specifying a #fragement")
	}

	// if we have no destFrag, we will use srcFrag
	if destFrag == "" {
		destFrag = srcFrag
	}

	var srcJson map[string]interface{}
	var destJson map[string]interface{}

	err = json.Unmarshal(srcJsonBuf, &srcJson)
	if err != nil {
		return err
	}

	err = json.Unmarshal(destJsonBuf, &destJson)
	if err != nil {
		return err
	}

	// reduce destJson if we are not merging
	if !merge {
		for k := range destJson {
			if destFrag != "" && strings.HasPrefix(k, destFrag+"/") {
				delete(destJson, k)
			} else if destFrag == "" && strings.Contains(k, "/") {
				// no destFrag we remove all in any folder
				delete(destJson, k)
			}
		}
	}

	// copy over relevant key/values
	for k, v := range srcJson {
		if (srcFrag != "" && strings.HasPrefix(k, srcFrag+"/")) ||
			(srcFrag == "" && strings.Contains(k, "/")) {
			nk := strings.TrimPrefix(k, srcFrag)
			nk = destFrag + nk
			destJson[nk] = v
		}
	}

	buf, err := p.postRemoteJson(destRemote, destJson, envelope, commitMsg, rev, force)

	if err != nil {
		return err
	}

	responseMap := map[string]interface{}{}

	err = json.Unmarshal(buf, &responseMap)

	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Successfully posted Revision %d (%s) to device id %s\n", int(responseMap["rev"].(float64)),
		responseMap["state-sha"].(string)[:8], responseMap["trail-id"])

	return nil
}
