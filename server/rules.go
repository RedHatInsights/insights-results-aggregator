package server

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// getRule returns rule with content for provided rule ID and rule error key
func (server *HTTPServer) getRule(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readRuleID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	errorKey, err := readErrorKey(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	ruleWithContent, err := server.Storage.GetRuleWithContent(ruleID, errorKey)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("rule", ruleWithContent))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// likeRule likes the rule for current user
func (server *HTTPServer) likeRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, types.UserVoteLike)
}

// dislikeRule dislikes the rule for current user
func (server *HTTPServer) dislikeRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, types.UserVoteDislike)
}

// resetVoteOnRule resets vote for the rule for current user
func (server *HTTPServer) resetVoteOnRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, types.UserVoteNone)
}

func (server *HTTPServer) voteOnRule(writer http.ResponseWriter, request *http.Request, userVote types.UserVote) {
	clusterID, ruleID, userID, err := server.readClusterRuleUserParams(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.checkUserClusterPermissions(writer, request, clusterID)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.Storage.VoteOnRule(clusterID, ruleID, userID, userVote)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) getVoteOnRule(writer http.ResponseWriter, request *http.Request) {
	clusterID, ruleID, userID, err := server.readClusterRuleUserParams(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.checkUserClusterPermissions(writer, request, clusterID)
	if err != nil {
		// everything has been handled already
		return
	}

	userFeedbackOnRule, err := server.Storage.GetUserFeedbackOnRule(clusterID, ruleID, userID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("vote", userFeedbackOnRule.UserVote))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// disableRuleForCluster disables a rule for specified cluster, excluding it from reports
func (server *HTTPServer) disableRuleForCluster(writer http.ResponseWriter, request *http.Request) {
	server.toggleRuleForCluster(writer, request, storage.RuleToggleDisable)
}

// enableRuleForCluster enables a previously disabled rule, showing it on reports again
func (server *HTTPServer) enableRuleForCluster(writer http.ResponseWriter, request *http.Request) {
	server.toggleRuleForCluster(writer, request, storage.RuleToggleEnable)
}

// toggleRuleForCluster contains shared functionality for enable/disable
func (server *HTTPServer) toggleRuleForCluster(writer http.ResponseWriter, request *http.Request, toggleRule storage.RuleToggle) {
	clusterID, ruleID, userID, err := server.readClusterRuleUserParams(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.checkUserClusterPermissions(writer, request, clusterID)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.Storage.ToggleRuleForCluster(clusterID, ruleID, userID, toggleRule)
	if err != nil {
		log.Error().Err(err).Msg("Unable to toggle rule for selected cluster")
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) createRule(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readRuleID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	var rule types.Rule

	err = json.NewDecoder(request.Body).Decode(&rule)
	if err != nil {
		if err == io.EOF {
			err = &NoBodyError{}
		}
		handleServerError(writer, err)
		return
	}

	rule.Module = ruleID

	err = server.Storage.CreateRule(rule)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData(
		"rule", rule,
	))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) deleteRule(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readRuleID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.Storage.DeleteRule(ruleID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) createRuleErrorKey(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readRuleID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	// it's gonna raise an error if rule does not exist
	_, err = server.Storage.GetRuleByID(ruleID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	errorKey, err := readErrorKey(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	var ruleErrorKey types.RuleErrorKey

	err = json.NewDecoder(request.Body).Decode(&ruleErrorKey)
	if err != nil {
		if err == io.EOF {
			err = &NoBodyError{}
		}
		handleServerError(writer, err)
		return
	}

	ruleErrorKey.RuleModule = ruleID
	ruleErrorKey.ErrorKey = errorKey

	err = server.Storage.CreateRuleErrorKey(ruleErrorKey)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData(
		"rule_error_key", ruleErrorKey,
	))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) deleteRuleErrorKey(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readRuleID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	errorKey, err := readErrorKey(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.Storage.DeleteRuleErrorKey(ruleID, errorKey)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// deleteRuleForClusterToggle is debug endpoint for deleting the record in the rule_cluster_toggle table
func (server *HTTPServer) deleteFromRuleClusterToggle(writer http.ResponseWriter, request *http.Request) {
	clusterID, ruleID, userID, err := server.readClusterRuleUserParams(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.checkUserClusterPermissions(writer, request, clusterID)
	if err != nil {
		// everything has been handled already
		return
	}

	err = server.Storage.DeleteFromRuleClusterToggle(clusterID, ruleID, userID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to delete from rule_cluster_toggle")
		handleServerError(writer, err)
		return
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func getTotalRuleCount(reportRules types.ReportRules) int {
	totalCount := len(reportRules.HitRules) +
		len(reportRules.SkippedRules) +
		len(reportRules.PassedRules)
	return totalCount
}

// getContentForRules returns the hit rules from the report, as well as total count of all rules (skipped, ..)
func (server *HTTPServer) getContentForRules(
	writer http.ResponseWriter,
	report types.ClusterReport,
) ([]types.RuleContentResponse, int, error) {
	var reportRules types.ReportRules

	err := json.Unmarshal([]byte(report), &reportRules)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse cluster report")
		handleServerError(writer, err)
		return nil, 0, err
	}

	totalRules := getTotalRuleCount(reportRules)

	hitRules, err := server.Storage.GetContentForRules(reportRules)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve rules content from database")
		handleServerError(writer, err)
		return nil, 0, err
	}

	return hitRules, totalRules, nil
}

// getUserVoteForRules returns user votes for defined list of report's IDs
func (server *HTTPServer) getUserVoteForRules(
	feedbacks map[types.RuleID]types.UserVote,
	rulesContent []types.RuleContentResponse,
) []types.RuleContentResponse {
	for i := range rulesContent {
		if vote, found := feedbacks[types.RuleID(rulesContent[i].RuleModule)]; found {
			rulesContent[i].UserVote = vote
		} else {
			rulesContent[i].UserVote = types.UserVoteNone
		}
	}
	return rulesContent
}
