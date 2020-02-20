/*
Copyright © 2020 Red Hat, Inc.

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

package server_test

import (
	"net/http"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/server"
)

var configAuth = server.Configuration{
	Address:   ":8080",
	APIPrefix: "/api/test/",
	Debug:     false,
}

func TestMissingAuthToken(t *testing.T) {
	server := server.New(configAuth, nil)
	req, err := http.NewRequest("GET", configAuth.APIPrefix+"organizations", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(server, req).Result()
	checkResponseCode(t, http.StatusForbidden, response.StatusCode)
}

func TestInvalidAuthToken(t *testing.T) {
	server := server.New(configAuth, nil)
	req, err := http.NewRequest("GET", configAuth.APIPrefix+"organizations", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("x-rh-identity", "123456qwerty")

	response := executeRequest(server, req).Result()
	checkResponseCode(t, http.StatusForbidden, response.StatusCode)
}

func TestInvalidJsonAuthToken(t *testing.T) {
	server := server.New(configAuth, nil)
	req, err := http.NewRequest("GET", configAuth.APIPrefix+"organizations", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("x-rh-identity", "aW52YWxpZCBqc29uCg==")

	response := executeRequest(server, req).Result()
	checkResponseCode(t, http.StatusForbidden, response.StatusCode)
}

func TestBadOrganizationID(t *testing.T) {
	server := server.New(configAuth, nil)
	req, err := http.NewRequest("GET", configAuth.APIPrefix+"organizations/12345/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("x-rh-identity", "eyJpZGVudGl0eSI6IHsiaW50ZXJuYWwiOiB7Im9yZ19pZCI6ICIxMjM0In19fQo=")

	response := executeRequest(server, req).Result()
	checkResponseCode(t, http.StatusForbidden, response.StatusCode)
}
