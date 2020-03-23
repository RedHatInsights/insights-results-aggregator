package server

import (
	"fmt"
	"regexp"
)

const (
	MainEndpoint                    = ""
	DeleteOrganizationsEndpoint     = "organizations/{organizations}"
	DeleteClustersEndpoint          = "clusters/{clusters}"
	OrganizationsEndpoint           = "organizations"
	ReportEndpoint                  = "report/{organization}/{cluster}"
	LikeRuleEndpoint                = "clusters/{cluster}/rules/{rule_id}/like"
	DislikeRuleEndpoint             = "clusters/{cluster}/rules/{rule_id}/dislike"
	ResetVoteOnRuleEndpoint         = "clusters/{cluster}/rules/{rule_id}/reset_vote"
	ClustersForOrganizationEndpoint = "organizations/{organization}/clusters"
	MetricsEndpoint                 = "metrics"
)

// MakeURLToEndpoint creates URL to endpoint, use constants from file endpoints.go
func MakeURLToEndpoint(apiPrefix, endpoint string, args ...interface{}) string {
	re := regexp.MustCompile(`\{[a-zA-Z_0-9]+\}`)
	endpoint = re.ReplaceAllString(endpoint, "%v")
	return apiPrefix + fmt.Sprintf(endpoint, args...)
}
