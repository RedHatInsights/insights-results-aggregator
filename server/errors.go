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
	operator_utils_types "github.com/RedHatInsights/insights-operator-utils/types"
)

type (
	// NoBodyError error meaning that client didn't provide body when it's required
	NoBodyError = operator_utils_types.NoBodyError
	// RouterMissingParamError missing parameter in request
	RouterMissingParamError = operator_utils_types.RouterMissingParamError
	// RouterParsingError parsing error, for example string when we expected integer
	RouterParsingError = operator_utils_types.RouterParsingError
	// UnauthorizedError means server can't authorize you, for example the token is missing or malformed
	UnauthorizedError = operator_utils_types.UnauthorizedError
	// ForbiddenError means you don't have permission to do a particular action,
	ForbiddenError = operator_utils_types.ForbiddenError
)

// handleServerError handles separate server errors and sends appropriate responses
var handleServerError = operator_utils_types.HandleServerError

// responseDataError is used as the error message when the responses functions return an error
const responseDataError = "Unexpected error during response data encoding"
