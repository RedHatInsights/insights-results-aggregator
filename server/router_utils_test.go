// Copyright 2020, 2021, 2022 Red Hat, Inc
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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

const (
	cluster1ID = "715e10eb-e6ac-49b3-bd72-61734c35b6fb"
	cluster2ID = "931f1495-7b16-4637-a41e-963e117bfd02"
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

	recorder := httptest.NewRecorder()

	_, successful := server.ReadClusterName(recorder, request)
	assert.False(t, successful)

	resp := recorder.Result()
	assert.NotNil(t, resp)

	body, err := ioutil.ReadAll(resp.Body)
	helpers.FailOnError(t, err)

	assert.Equal(t, `{"status":"Missing required param from request: cluster"}`, strings.TrimSpace(string(body)))
}

func TestReadOrganizationIDMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, successful := server.ReadOrganizationID(httptest.NewRecorder(), request, false)
	assert.False(t, successful)
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

	_, successful := server.ReadClusterNames(httptest.NewRecorder(), request)
	assert.False(t, successful)
}

func TestReadOrganizationIDsMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, successful := server.ReadOrganizationIDs(httptest.NewRecorder(), request)
	assert.False(t, successful)
}

func TestReadRuleIDMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	recorder := httptest.NewRecorder()

	_, successful := server.ReadRuleID(recorder, request)
	assert.False(t, successful)

	resp := recorder.Result()
	assert.NotNil(t, resp)

	body, err := ioutil.ReadAll(resp.Body)
	helpers.FailOnError(t, err)

	assert.Equal(t, `{"status":"Missing required param from request: rule_id"}`, strings.TrimSpace(string(body)))
}

// TestReadClusterListFromPathMissingClusterList function checks if missing
// cluster list in path is detected and processed correctly by function
// ReadClusterListFromPath.
func TestReadClusterListFromPathMissingClusterList(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	// try to read list of clusters from path
	_, successful := server.ReadClusterListFromPath(httptest.NewRecorder(), request)

	// missing list means that the read operation should fail
	assert.False(t, successful)
}

// TestReadClusterListFromPathEmptyClusterList function checks if empty cluster
// list in path is detected and processed correctly by function
// ReadClusterListFromPath.
func TestReadClusterListFromPathEmptyClusterList(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, http.MethodGet, "", nil, map[string]string{
		"cluster_list": "",
	})

	// try to read list of clusters from path
	_, successful := server.ReadClusterListFromPath(httptest.NewRecorder(), request)

	// empty list means that the read operation should fail
	assert.False(t, successful)
}

// TestReadClusterListFromPathOneCluster function checks if list with one
// cluster ID is processed correctly by function ReadClusterListFromPath.
func TestReadClusterListFromPathOneCluster(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, http.MethodGet, "", nil, map[string]string{
		"cluster_list": cluster1ID,
	})

	// try to read list of clusters from path
	list, successful := server.ReadClusterListFromPath(httptest.NewRecorder(), request)

	// cluster list exists so the read operation should not fail
	assert.True(t, successful)

	// we expect do get list with one cluster ID
	assert.ElementsMatch(t, list, []string{cluster1ID})
}

// TestReadClusterListFromPathTwoClusters function checks if list with two
// cluster IDs is processed correctly by function ReadClusterListFromPath.
func TestReadClusterListFromPathTwoClusters(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, http.MethodGet, "", nil, map[string]string{
		"cluster_list": fmt.Sprintf("%v,%v", cluster1ID, cluster2ID),
	})

	// try to read list of clusters from path
	list, successful := server.ReadClusterListFromPath(httptest.NewRecorder(), request)

	// cluster list exists so the read operation should not fail
	assert.True(t, successful)

	// we expect do get list with two cluster IDs
	assert.ElementsMatch(t, list, []string{cluster1ID, cluster2ID})
}

// TestReadClusterListFromBodyNoJSON function checks if reading list of
// clusters from empty request body is detected properly by function
// ReadClusterListFromBody.
func TestReadClusterListFromBodyNoJSON(t *testing.T) {
	request, err := http.NewRequest(
		http.MethodGet,
		"",
		strings.NewReader(""),
	)
	helpers.FailOnError(t, err)

	// try to read list of clusters from path
	_, successful := server.ReadClusterListFromBody(httptest.NewRecorder(), request)

	// the read should fail because of empty request body
	assert.False(t, successful)
}

// TestReadClusterListFromBodyCorrectJSON function checks if reading list of
// clusters from correct request body containing JSON data is done correctly by
// function ReadClusterListFromBody.
func TestReadClusterListFromBodyCorrectJSON(t *testing.T) {
	request, err := http.NewRequest(
		http.MethodGet,
		"",
		strings.NewReader(fmt.Sprintf(`{"clusters": ["%v","%v"]}`, cluster1ID, cluster2ID)),
	)
	helpers.FailOnError(t, err)

	// try to read list of clusters from path
	list, successful := server.ReadClusterListFromBody(httptest.NewRecorder(), request)

	// cluster list exists so the call should not fail
	assert.True(t, successful)

	// we expect do get list with two cluster IDs
	assert.ElementsMatch(t, list, []string{cluster1ID, cluster2ID})
}

// TestReadClusterListFromBodyWrongJSON function checks if reading list of
// clusters from request body with improper format is processed correctly by
// function ReadClusterListFromBody.
func TestReadClusterListFromBodyWrongJSON(t *testing.T) {
	request, err := http.NewRequest(
		http.MethodGet,
		"",
		strings.NewReader("this-is-not-json"),
	)
	helpers.FailOnError(t, err)

	// try to read list of clusters from path
	_, successful := server.ReadClusterListFromBody(httptest.NewRecorder(), request)

	// the read should fail because of broken JSON
	assert.False(t, successful)
}
