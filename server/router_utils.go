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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// RouterMissingParamError missing parameter in URL
type RouterMissingParamError struct {
	paramName string
}

func (e *RouterMissingParamError) Error() string {
	return fmt.Sprintf("missing param %v", e.paramName)
}

// RouterParsingError parsing error, for example string when we expected integer
type RouterParsingError struct {
	paramName  string
	paramValue interface{}
	errString  string
}

func (e *RouterParsingError) Error() string {
	return fmt.Sprintf(
		"Error during parsing param %v with value %v. Error: %v",
		e.paramName, e.paramValue, e.errString,
	)
}

// getRouterParam retrieves parameter from URL like `/organization/{org_id}`
func getRouterParam(request *http.Request, paramName string) (string, error) {
	value, found := mux.Vars(request)[paramName]
	if !found {
		return "", &RouterMissingParamError{paramName: paramName}
	}

	return value, nil
}

// getRouterPositiveIntParam retrieves parameter from URL like `/organization/{org_id}`
// and check it for being valid and positive integer, otherwise returns error
func getRouterPositiveIntParam(request *http.Request, paramName string) (uint64, error) {
	value, err := getRouterParam(request, paramName)
	if err != nil {
		return 0, err
	}

	uintValue, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, &RouterParsingError{
			paramName: paramName, paramValue: value, errString: "unsigned integer expected",
		}
	}

	if uintValue == 0 {
		return 0, &RouterParsingError{
			paramName: paramName, paramValue: value, errString: "positive value expected",
		}
	}

	return uintValue, nil
}

// validateClusterName checks that the cluster name is a valid UUID.
// Converted custer name is returned if everything is okay, otherwise an error is returned.
func validateClusterName(writer http.ResponseWriter, clusterName string) (types.ClusterName, error) {
	if _, err := uuid.Parse(clusterName); err != nil {
		message := fmt.Sprintf("invalid cluster name format: '%s'", clusterName)

		log.Error().Err(err).Msg(message)
		responses.SendInternalServerError(writer, message)

		return "", errors.New(message)
	}

	return types.ClusterName(clusterName), nil
}

// splitRequestParamArray takes a single HTTP request parameter and splits it
// into a slice of strings. This assumes that the parameter is a comma-separated array.
func splitRequestParamArray(arrayParam string) []string {
	return strings.Split(arrayParam, ",")
}

func handleOrgIDError(writer http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("Error getting organization ID from request")

	if _, ok := err.(*RouterParsingError); ok {
		responses.Send(http.StatusBadRequest, writer, err.Error())
	} else {
		responses.Send(http.StatusInternalServerError, writer, err.Error())
	}
}

func handleClusterNameError(writer http.ResponseWriter, err error) {
	message := fmt.Sprintf("Cluster name is not provided %v", err.Error())
	log.Error().Msg(message)

	// query parameter 'cluster' can't be found in request, which might be caused by issue in Gorilla mux
	// (not on client side)
	responses.SendInternalServerError(writer, message)
}

// readClusterName retrieves cluster name from request
// if it's not possible, it writes http error to the writer and returns error
func readClusterName(writer http.ResponseWriter, request *http.Request) (types.ClusterName, error) {
	clusterName, err := getRouterParam(request, "cluster")
	if err != nil {
		handleClusterNameError(writer, err)
		return "", err
	}

	return validateClusterName(writer, clusterName)
}

// readOrganizationID retrieves organization id from request
// if it's not possible, it writes http error to the writer and returns error
func readOrganizationID(writer http.ResponseWriter, request *http.Request) (types.OrgID, error) {
	organizationID, err := getRouterPositiveIntParam(request, "organization")
	if err != nil {
		handleOrgIDError(writer, err)
		return 0, err
	}

	return types.OrgID(organizationID), nil
}

// readClusterNames does the same as `readClusterName`, except for multiple clusters.
func readClusterNames(writer http.ResponseWriter, request *http.Request) ([]types.ClusterName, error) {
	clusterNamesParam, err := getRouterParam(request, "clusters")
	if err != nil {
		message := fmt.Sprintf("Cluster names are not provided %v", err.Error())
		log.Error().Msg(message)

		// See `readClusterName`.
		responses.SendInternalServerError(writer, message)

		return []types.ClusterName{}, err
	}

	clusterNamesConverted := []types.ClusterName{}
	for _, clusterName := range splitRequestParamArray(clusterNamesParam) {
		convertedName, err := validateClusterName(writer, clusterName)
		if err != nil {
			return []types.ClusterName{}, err
		}

		clusterNamesConverted = append(clusterNamesConverted, convertedName)
	}

	return clusterNamesConverted, nil
}

// readOrganizationIDs does the same as `readOrganizationID`, except for multiple organizations.
func readOrganizationIDs(writer http.ResponseWriter, request *http.Request) ([]types.OrgID, error) {
	organizationsParam, err := getRouterParam(request, "organizations")
	if err != nil {
		handleOrgIDError(writer, err)
		return []types.OrgID{}, err
	}

	organizationsConverted := []types.OrgID{}
	for _, orgStr := range splitRequestParamArray(organizationsParam) {
		orgInt, err := strconv.ParseUint(orgStr, 10, 64)
		if err != nil {
			return []types.OrgID{}, err
		}
		organizationsConverted = append(organizationsConverted, types.OrgID(orgInt))
	}

	return organizationsConverted, nil
}
