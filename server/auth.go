// Auth implementation based on JWT

/*
Copyright © 2019, 2020, 2021, 2022 Red Hat, Inc.

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
	"strings"

	"github.com/RedHatInsights/insights-operator-utils/collections"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

const (
	// #nosec G101
	malformedTokenMessage = "Malformed authentication token"
)

// Internal contains information about organization ID
type Internal = types.Internal

// Identity contains internal user info
type Identity = types.Identity

// Token is x-rh-identity struct
type Token = types.Token

// JWTPayload is structure that contain data from parsed JWT token
type JWTPayload = types.JWTPayload

// Authentication middleware for checking auth rights
func (server *HTTPServer) Authentication(next http.Handler, noAuthURLs []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// for specific URLs it is ok to not use auth. mechanisms at all
		// this is specific to OpenAPI JSON response and for all OPTION HTTP methods
		if collections.StringInSlice(r.RequestURI, noAuthURLs) || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		token, err := server.getAuthTokenHeader(w, r)
		if err != nil {
			log.Error().Err(err).Msg(err.Error())
			handleServerError(w, err)
			return
		}

		// decode auth. token to JSON string
		decoded, err := base64.StdEncoding.DecodeString(token)

		// if token is malformed return HTTP code 403 to client
		if err != nil {
			// malformed token, returns with HTTP code 403 as usual
			log.Error().Err(err).Msg(malformedTokenMessage)
			handleServerError(w, &UnauthorizedError{ErrString: malformedTokenMessage})
			return
		}

		tk := &types.Token{}
		tkV2 := &types.TokenV2{}

		// if we took JWT token, it has different structure than x-rh-identity
		if server.Config.AuthType == "jwt" {
			jwtPayload := &types.JWTPayload{}
			err = json.Unmarshal(decoded, jwtPayload)
			if err != nil {
				// malformed token, returns with HTTP code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &UnauthorizedError{ErrString: malformedTokenMessage})
				return
			}
			// Map JWT token to inner token
			tk.Identity = types.Identity{
				AccountNumber: jwtPayload.AccountNumber,
				Internal: types.Internal{
					OrgID: jwtPayload.OrgID,
				},
			}
		} else {
			// auth type is xrh (x-rh-identity header)

			// unmarshal new token structure (org_id on top level)
			err = json.Unmarshal(decoded, tkV2)
			if err != nil {
				// malformed token, returns with HTTP code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &UnauthorizedError{ErrString: malformedTokenMessage})
				return
			}

			// unmarshal old token structure (org_id nested) too
			err = json.Unmarshal(decoded, tk)
			if err != nil {
				// malformed token, returns with HTTP code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &UnauthorizedError{ErrString: malformedTokenMessage})
				return
			}
		}

		if tkV2.IdentityV2.OrgID != 0 {
			log.Info().Msg("org_id found on top level in token structure (new format)")
			// fill in old types.Token because many places in smart-proxy and aggregator rely on it.
			tk.Identity.Internal.OrgID = tkV2.IdentityV2.OrgID
		} else {
			log.Error().Msg("org_id not found on top level in token structure (old format)")
		}

		if tk.Identity.AccountNumber == "" || tk.Identity.Internal.OrgID == 0 {
			msg := fmt.Sprintf("error retrieving requester data from token. org_id [%v], account_number [%v], user data [%+v]",
				tk.Identity.Internal.OrgID,
				tk.Identity.AccountNumber,
				tkV2.IdentityV2.User,
			)
			log.Error().Msg(msg)
			handleServerError(w, &UnauthorizedError{ErrString: msg})
			return
		}

		// Everything went well, proceed with the request and set the
		// caller to the user retrieved from the parsed token
		ctx := context.WithValue(r.Context(), types.ContextKeyUser, tk.Identity)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// GetCurrentUserID retrieves current user's id from request
func (server *HTTPServer) GetCurrentUserID(request *http.Request) (types.UserID, error) {
	i := request.Context().Value(types.ContextKeyUser)

	if i == nil {
		log.Error().Msgf("user id was not found in request's context")
		return "", &UnauthorizedError{ErrString: "user id is not provided"}
	}

	identity, ok := i.(Identity)
	if !ok {
		return "", fmt.Errorf("contextKeyUser has wrong type")
	}

	return identity.User.UserID, nil
}

func (server *HTTPServer) getAuthTokenHeader(w http.ResponseWriter, r *http.Request) (string, error) {
	var tokenHeader string
	// In case of testing on local machine we don't take x-rh-identity header, but instead Authorization with JWT token in it
	if server.Config.AuthType == "jwt" {
		tokenHeader = r.Header.Get("Authorization") // Grab the token from the header
		splitted := strings.Split(tokenHeader, " ") // The token normally comes in format `Bearer {token-body}`, we check if the retrieved token matched this requirement
		if len(splitted) != 2 {
			const message = "Invalid/Malformed auth token"
			return "", &UnauthorizedError{ErrString: message}
		}

		// Here we take JWT token which include 3 parts, we need only second one
		splitted = strings.Split(splitted[1], ".")
		tokenHeader = splitted[1]
	} else {
		tokenHeader = r.Header.Get("x-rh-identity") // Grab the token from the header
	}

	if tokenHeader == "" {
		const message = "Missing auth token"
		return "", &UnauthorizedError{ErrString: message}
	}

	return tokenHeader, nil
}
