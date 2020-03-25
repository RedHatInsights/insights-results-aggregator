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
	"encoding/json"
	"fmt"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog/log"
	"net/http"
	"strings"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

type contextKey string

const (
	// ContextKeyUser is a constant for user authentication token in request
	ContextKeyUser        = contextKey("user")
	malformedTokenMessage = "Malformed authentication token"
)

// Internal contains information about organization ID
type Internal struct {
	OrgID string `json:"org_id"`
}

// Identity contains internal user info
type Identity struct {
	AccountNumber types.UserID `json:"account_number"`
	Internal      Internal     `json:"internal"`
}

// Token is x-rh-identity struct
type Token struct {
	Identity Identity `json:"identity"`
}

// JWTPayload is structure that contain data from parsed JWT token
type JWTPayload struct {
	AccountNumber types.UserID `json:"account_number"`
	OrgID         string       `json:"org_id"`
}

// Authentication middleware for checking auth rights
func (server *HTTPServer) Authentication(next http.Handler, noAuthURLs []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// for specific URLs it is ok to not use auth. mechanisms at all
		if stringInSlice(r.RequestURI, noAuthURLs) {
			next.ServeHTTP(w, r)
			return
		}
		var tokenHeader string
		// In case of testing on local machine we don't take x-rh-identity header, but instead Authorization with JWT token in it
		if server.Config.Debug {
			tokenHeader = r.Header.Get("Authorization") //Grab the token from the header
			splitted := strings.Split(tokenHeader, " ") //The token normally comes in format `Bearer {token-body}`, we check if the retrieved token matched this requirement
			if len(splitted) != 2 {
				const message = "Invalid/Malformed auth token"
				log.Error().Msg(message)
				handleServerError(w, &AuthenticationError{errString: message})
				return
			}

			// Here we take JWT token which include 3 parts, we need only second one
			splitted = strings.Split(splitted[1], ".")
			tokenHeader = splitted[1]
		} else {
			tokenHeader = r.Header.Get("x-rh-identity") //Grab the token from the header
		}

		// Token is missing, SHOULD RETURN with error code 403 Unauthorized - changes in utils necessary
		// TODO: Change SendUnauthorized in utils to accept string instead of map interface and change here
		if tokenHeader == "" {
			const message = "Missing auth token"
			log.Error().Msg(message)
			handleServerError(w, &AuthenticationError{errString: message})
			return
		}

		decoded, err := jwt.DecodeSegment(tokenHeader) // Decode token to JSON string
		if err != nil {                                // Malformed token, returns with http code 403 as usual
			log.Error().Err(err).Msg(malformedTokenMessage)
			handleServerError(w, &AuthenticationError{errString: malformedTokenMessage})
			return
		}

		tk := &Token{}

		// If we took JWT token, it has different structure then x-rh-identity
		if server.Config.Debug {
			jwt := &JWTPayload{}
			err = json.Unmarshal([]byte(decoded), jwt)
			if err != nil { //Malformed token, returns with http code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &AuthenticationError{errString: malformedTokenMessage})
				return
			}
			// Map JWT token to inner token
			tk.Identity = Identity{AccountNumber: jwt.AccountNumber, Internal: Internal{OrgID: jwt.OrgID}}
		} else {
			err = json.Unmarshal([]byte(decoded), tk)

			if err != nil { //Malformed token, returns with http code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &AuthenticationError{errString: malformedTokenMessage})
				return
			}
		}

		// Everything went well, proceed with the request and set the caller to the user retrieved from the parsed token
		ctx := context.WithValue(r.Context(), ContextKeyUser, tk.Identity)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetCurrentUserID retrieves current user's id from request
func (server *HTTPServer) GetCurrentUserID(request *http.Request) (types.UserID, error) {
	i := request.Context().Value(ContextKeyUser)

	identity, ok := i.(Identity)
	if !ok {
		return "", fmt.Errorf("contextKeyUser has wrong type")
	}

	return identity.AccountNumber, nil
}
