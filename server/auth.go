// Auth implementation based on JWT

/*
Copyright © 2019, 2020 Red Hat, Inc.

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

package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/insights-results-aggregator/types"

	"github.com/RedHatInsights/insights-operator-utils/responses"
)

type contextKey string

const (
	contextKeyUser = contextKey("user")
)

// Internal contains information about organization ID
type Internal struct {
	OrgID string `json:"org_id"`
}

// Identity contains internal user info
type Identity struct {
	AccountNumber string   `json:"account_number"`
	Internal      Internal `json:"internal"`
}

// Token is x-rh-identity struct
type Token struct {
	Identity Identity `json:"identity"`
}

// Authentication middleware for checking auth rights
func (server *HTTPServer) Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenHeader := r.Header.Get("x-rh-identity") //Grab the token from the header

		if tokenHeader == "" { //Token is missing, returns with error code 403 Unauthorized
			responses.SendForbidden(w, "Missing auth token")
			return
		}

		decoded, err := base64.StdEncoding.DecodeString(tokenHeader) //Decode token to JSON string
		if err != nil {                                              //Malformed token, returns with http code 403 as usual
			responses.SendForbidden(w, "Malformed authentication token")
			return
		}

		tk := &Token{}
		err = json.Unmarshal(decoded, tk)

		if err != nil { //Malformed token, returns with http code 403 as usual
			responses.SendForbidden(w, "Malformed authentication token")
			return
		}

		//Everything went well, proceed with the request and set the caller to the user retrieved from the parsed token
		ctx := context.WithValue(r.Context(), contextKeyUser, tk.Identity)
		r = r.WithContext(ctx)
		// Proceed to proxy
		next.ServeHTTP(w, r)
	})
}

// GetCurrentUserID retrieves current user's id from request
func (server *HTTPServer) GetCurrentUserID(request *http.Request) (types.UserID, error) {
	i := request.Context().Value(contextKeyUser)

	identity, ok := i.(Identity)
	if !ok {
		return "", fmt.Errorf("contextKeyUser has wrong type")
	}

	return types.UserID(identity.AccountNumber), nil
}
