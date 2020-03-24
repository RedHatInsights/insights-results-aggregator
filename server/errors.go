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
	"fmt"
	"net/http"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

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

// handleServerError handles separate parsing errors and sends appropriate responses
func handleServerError(writer http.ResponseWriter, err error) {
	switch err := err.(type) {
	case *RouterMissingParamError:
		responses.SendError(writer, err.Error())
	case *RouterParsingError:
		responses.SendError(writer, err.Error())
	case *storage.ItemNotFoundError:
		responses.SendNotFound(writer, err.Error())
	default:
		responses.SendInternalServerError(writer, err.Error())
	}
}
