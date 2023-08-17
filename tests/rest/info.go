/*
Copyright © 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"encoding/json"

	"github.com/verdverm/frisby"
)

// InfoResponse represents response from /info endpoint
type InfoResponse struct {
	Info   map[string]string `json:"info"`
	Status string            `json:"status"`
}

// checkInfoEndpoint performs elementary checks for /info endpoint
func checkInfoEndpoint() {
	var expectedInfoKeys = []string{
		"BuildBranch",
		"BuildCommit",
		"BuildTime",
		"BuildVersion",
		"DB_version",
		"UtilsVersion",
	}

	f := frisby.Create("Check the /info endpoint").Get(apiURL + "info")
	setAuthHeaderForOrganization(f, knownOrganizations[0])
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, "application/json; charset=utf-8")

	// check the response
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		response := InfoResponse{}
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}
		if response.Status != "ok" {
			f.AddError("Expecting 'status' to be set to 'ok'")
		}
		if len(response.Info) == 0 {
			f.AddError("Info node is empty")
		}
		for _, expectedKey := range expectedInfoKeys {
			_, found := response.Info[expectedKey]
			if !found {
				f.AddError("Info node does not contain key " + expectedKey)
			}
		}
	}
	f.PrintReport()
}
