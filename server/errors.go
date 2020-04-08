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

package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// responseDataError is used as the error message when the responses functions return an error
const responseDataError = "Unexpected error during response data encoding"

// RouterMissingParamError missing parameter in request
type RouterMissingParamError struct {
	paramName string
}

func (e *RouterMissingParamError) Error() string {
	return fmt.Sprintf("Missing required param from request: %v", e.paramName)
}

// RouterParsingError parsing error, for example string when we expected integer
type RouterParsingError struct {
	paramName  string
	paramValue interface{}
	errString  string
}

func (e *RouterParsingError) Error() string {
	return fmt.Sprintf(
		"Error during parsing param '%v' with value '%v'. Error: '%v'",
		e.paramName, e.paramValue, e.errString,
	)
}

// AuthenticationError happens during auth problems, for example malformed token
type AuthenticationError struct {
	errString string
}

func (e *AuthenticationError) Error() string {
	return e.errString
}

// handleServerError handles separate server errors and sends appropriate responses
func handleServerError(writer http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("handleServerError()")

	var respErr error

	switch err := err.(type) {
	case *RouterMissingParamError:
		respErr = responses.SendError(writer, err.Error())
	case *RouterParsingError:
		respErr = responses.SendError(writer, err.Error())
	case *storage.ItemNotFoundError:
		respErr = responses.SendNotFound(writer, err.Error())
	case *AuthenticationError:
		respErr = responses.SendForbidden(writer, err.Error())
	case *json.SyntaxError:
		respErr = responses.Send(http.StatusBadRequest, writer, err.Error())
	default:
		respErr = responses.SendInternalServerError(writer, "Internal Server Error")
	}

	if respErr != nil {
		log.Error().Err(respErr).Msg(responseDataError)
	}
}
