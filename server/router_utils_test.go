// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestGetRouterIntParamMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "organizations//clusters", nil)
	helpers.FailOnError(t, err)

	_, err = server.GetRouterPositiveIntParam(request, "test")
	assert.EqualError(t, err, "Missing required param from request: test")
}

func TestReadClusterNameMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadClusterName(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "Missing required param from request: cluster")
}

func TestReadOrganizationIDMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadOrganizationID(httptest.NewRecorder(), request, false)
	assert.EqualError(t, err, "Missing required param from request: organization")
}

func mustGetRequestWithMuxVars(
	t *testing.T,
	method string,
	url string,
	body io.Reader,
	vars map[string]string,
) *http.Request {
	request, err := http.NewRequest(method, url, body)
	helpers.FailOnError(t, err)

	request = mux.SetURLVars(request, vars)

	return request
}

func TestGetRouterIntParamNonIntError(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, http.MethodGet, "", nil, map[string]string{
		"id": "non int",
	})

	_, err := server.GetRouterPositiveIntParam(request, "id")
	assert.EqualError(
		t,
		err,
		"Error during parsing param 'id' with value 'non int'. Error: 'unsigned integer expected'",
	)
}

func TestGetRouterIntParamOK(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, http.MethodGet, "", nil, map[string]string{
		"id": "99",
	})

	id, err := server.GetRouterPositiveIntParam(request, "id")
	helpers.FailOnError(t, err)

	assert.Equal(t, uint64(99), id)
}

func TestGetRouterPositiveIntParamZeroError(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, http.MethodGet, "", nil, map[string]string{
		"id": "0",
	})

	_, err := server.GetRouterPositiveIntParam(request, "id")
	assert.EqualError(t, err, "Error during parsing param 'id' with value '0'. Error: 'positive value expected'")
}

func TestReadClusterNamesMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadClusterNames(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "Missing required param from request: clusters")
}

func TestReadOrganizationIDsMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadOrganizationIDs(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "Missing required param from request: organizations")
}

func TestReadRuleIDMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadRuleID(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "Missing required param from request: rule_id")
}
