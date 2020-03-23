package helpers

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/types"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// TODO: add docs
type APIRequest struct {
	Method       string
	Endpoint     string
	EndpointArgs []interface{}
	Body         string
	UserID       types.UserID
	XRHIdentity  string
}

type APIResponse struct {
	StatusCode  int
	Body        string
	BodyChecker func(t *testing.T, expected, got string)
}

// TODO: fix the doc
// AssertHTTPCodeAndBodyJSON creates new server with provided mockStorage
// (which you can keep nil so it will be created automatically)
// sends http request and checks if code and body is equal to the provided one
func AssertAPIRequest(
	t *testing.T,
	mockStorage storage.Storage,
	serverConfig *server.Configuration,
	request *APIRequest,
	expectedResponse *APIResponse,
) {
	if mockStorage == nil {
		mockStorage = MustGetMockStorage(t, true)
		defer MustCloseStorage(t, mockStorage)
	}

	testServer := server.New(*serverConfig, mockStorage)

	url := server.MakeURLToEndpoint(serverConfig.APIPrefix, request.Endpoint, request.EndpointArgs...)

	req, err := http.NewRequest(request.Method, url, strings.NewReader(request.Body))
	FailOnError(t, err)

	// authorize user
	if request.UserID != types.UserID(0) {
		identity := server.Identity{
			AccountNumber: request.UserID,
		}
		req = req.WithContext(context.WithValue(req.Context(), server.ContextKeyUser, identity))
	}

	if len(request.XRHIdentity) != 0 {
		req.Header.Set("x-rh-identity", request.XRHIdentity)
	}

	response := ExecuteRequest(testServer, req, serverConfig).Result()

	if expectedResponse.StatusCode != 0 {
		assert.Equal(t, expectedResponse.StatusCode, response.StatusCode, "Expected different status code")
	}
	if expectedResponse.BodyChecker != nil {
		bodyBytes, err := ioutil.ReadAll(response.Body)
		FailOnError(t, err)

		expectedResponse.BodyChecker(t, expectedResponse.Body, string(bodyBytes))
	} else if len(expectedResponse.Body) != 0 {
		CheckResponseBodyJSON(t, expectedResponse.Body, response.Body)
	}
}

func ExecuteRequest(testServer *server.HTTPServer, req *http.Request, config *server.Configuration) *httptest.ResponseRecorder {
	router := testServer.Initialize(config.Address)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

// CheckResponseBodyJSON checks if body is the same json as in expected
// (ignores whitespaces, newlines, etc)
// also validates both expected and body to be a valid json
func CheckResponseBodyJSON(t *testing.T, expectedJSON string, body io.ReadCloser) {
	result, err := ioutil.ReadAll(body)
	FailOnError(t, err)

	AssertStringsAreEqualJSON(t, expectedJSON, string(result))
}
