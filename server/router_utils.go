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
	"regexp"
	"strings"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var (
	readRuleID                = httputils.ReadRuleID
	readErrorKey              = httputils.ReadErrorKey
	getRouterParam            = httputils.GetRouterParam
	getRouterPositiveIntParam = httputils.GetRouterPositiveIntParam
	validateClusterName       = httputils.ValidateClusterName
	splitRequestParamArray    = httputils.SplitRequestParamArray
	handleOrgIDError          = httputils.HandleOrgIDError
	readClusterName           = httputils.ReadClusterName
	readOrganizationID        = httputils.ReadOrganizationID
	checkPermissions          = httputils.CheckPermissions
	readClusterNames          = httputils.ReadClusterNames
	readOrganizationIDs       = httputils.ReadOrganizationIDs
)

// readUserID retrieves user_id from request
// if it's not possible, it writes http error to the writer and returns false
func readUserID(writer http.ResponseWriter, request *http.Request) (types.UserID, bool) {
	userID, err := getRouterParam(request, "user_id")
	if err != nil {
		handleServerError(writer, err)
		return "", false
	}

	userID = strings.TrimSpace(userID)
	if len(userID) == 0 {
		handleServerError(writer, &RouterMissingParamError{ParamName: "user_id"})
		return "", false
	}

	return types.UserID(userID), true
}

// readOrgID retrieves org_id from request
// if it's not possible, it writes http error to the writer and returns false
func readOrgID(writer http.ResponseWriter, request *http.Request) (types.OrgID, bool) {
	orgID, err := getRouterPositiveIntParam(request, "org_id")
	if err != nil {
		handleServerError(writer, err)
		return 0, false
	}

	return types.OrgID(orgID), true
}

func readRuleIDWithErrorKey(writer http.ResponseWriter, request *http.Request) (types.RuleID, types.ErrorKey, bool) {
	ruleIDWithErrorKey, err := getRouterParam(request, "rule_id")
	if err != nil {
		const message = "unable to get rule id"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return types.RuleID(0), types.ErrorKey(0), false
	}

	splitedRuleID := strings.Split(string(ruleIDWithErrorKey), "|")

	if len(splitedRuleID) != 2 {
		err = fmt.Errorf("invalid rule ID, it must contain only rule ID and error key separated by |")
		log.Error().Err(err)
		handleServerError(writer, &RouterParsingError{
			ParamName:  "rule_id",
			ParamValue: ruleIDWithErrorKey,
			ErrString:  err.Error(),
		})
		return types.RuleID(0), types.ErrorKey(0), false
	}

	IDValidator := regexp.MustCompile(`^[a-zA-Z_0-9.]+$`)

	isRuleIDValid := IDValidator.Match([]byte(splitedRuleID[0]))
	isErrorKeyValid := IDValidator.Match([]byte(splitedRuleID[1]))

	if !isRuleIDValid || !isErrorKeyValid {
		err = fmt.Errorf("invalid rule ID, each part of ID must contain only from latin characters, number, underscores or dots")
		log.Error().Err(err)
		handleServerError(writer, &RouterParsingError{
			ParamName:  "rule_id",
			ParamValue: ruleIDWithErrorKey,
			ErrString:  err.Error(),
		})
		return types.RuleID(0), types.ErrorKey(0), false
	}

	return types.RuleID(splitedRuleID[0]), types.ErrorKey(splitedRuleID[1]), true
}

// readClusterRuleUserParams gets cluster_name, rule_id and user_id from current request
func (server *HTTPServer) readClusterRuleUserParams(
	writer http.ResponseWriter, request *http.Request,
) (types.ClusterName, types.RuleID, types.UserID, bool) {
	clusterID, successful := readClusterName(writer, request)
	if !successful {
		return "", "", "", false
	}

	ruleID, successful := readRuleID(writer, request)
	if !successful {
		return "", "", "", false
	}

	userID, successful := readUserID(writer, request)
	if !successful {
		// everything has been handled already
		return "", "", "", false
	}

	clusterExists, err := server.Storage.DoesClusterExist(clusterID)
	if err != nil {
		handleServerError(writer, err)
		return "", "", "", false
	}
	if !clusterExists {
		handleServerError(writer, &types.ItemNotFoundError{ItemID: clusterID})
		return "", "", "", false
	}

	return clusterID, ruleID, userID, true
}
