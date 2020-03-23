package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestGetRouterIntParamMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "organizations//clusters", nil)
	helpers.FailOnError(t, err)

	_, err = server.GetRouterPositiveIntParam(request, "test")
	assert.EqualError(t, err, "missing param test")
}

func TestReadClusterNameMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadClusterName(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "missing param cluster")
}

func TestReadOrganizationIDMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadOrganizationID(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "missing param organization")
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
		"Error during parsing param id with value non int. Error: unsigned integer expected",
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
	assert.EqualError(t, err, "Error during parsing param id with value 0. Error: positive value expected")
}

func TestReadClusterNamesMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadClusterNames(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "missing param clusters")
}

func TestReadOrganizationIDsMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadOrganizationIDs(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "missing param organizations")
}

func TestReadRuleIDMissing(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "", nil)
	helpers.FailOnError(t, err)

	_, err = server.ReadRuleID(httptest.NewRecorder(), request)
	assert.EqualError(t, err, "missing param rule_id")
}
