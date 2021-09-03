// Copyright 2021 Red Hat, Inc
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

package amsclient

import (
	"fmt"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// AMSClient allow us to interact the AMS API
type AMSClient struct {
	connection *sdk.Connection
}

// DefaultClient is the AMSClient used by default when no other is specified
var DefaultClient *AMSClient

// NewAMSClient creates an AMSClient from the configuration
func NewAMSClient(conf Configuration) (*AMSClient, error) {
	conn, err := sdk.NewConnectionBuilder().
		URL(conf.URL).
		Tokens(conf.Token).
		Build()

	if err != nil {
		log.Error().Err(err).Msg("Unable to build the connection to AMS API")
		return nil, err
	}

	return &AMSClient{
		connection: conn,
	}, nil
}

// func initDefaultClient() {
// 	if DefaultClient != nil {
// 		return
// 	}

// 	DefaultClient, err := NewAMSClient(conf.Configuration.AMSClientConf)
// 	if err != nil {
// 		log.Error().Err(err).Msg("Cannot init the default client for AMS API")
// 	}
// }

// GetClustersForOrganization retrieves the clusters for a given organization using the default client
func (c *AMSClient) GetClustersForOrganization(orgID types.OrgID) []types.ClusterName {
	var retval []types.ClusterName = []types.ClusterName{}

	internalOrgID, err := c.getInternalOrgIDFromExternal(orgID)
	if err != nil {
		return retval
	}

	subscriptionClient := c.connection.AccountsMgmt().V1().Subscriptions()
	response, err := subscriptionClient.List().Search(fmt.Sprintf("organization_id is '%s'", internalOrgID)).Send()
	if err != nil {
		return retval
	}

	for _, item := range response.Items().Slice() {
		clusterID, ok := item.GetExternalClusterID()
		item.GetOrganizationID()
		if !ok {
			continue
		}
		retval = append(retval, types.ClusterName(clusterID))
	}

	return retval
}

func (c *AMSClient) getInternalOrgIDFromExternal(orgID types.OrgID) (string, error) {
	orgsClient := c.connection.AccountsMgmt().V1().Organizations()
	response, err := orgsClient.List().Search(fmt.Sprintf("external_id = %d", orgID)).Send()
	if err != nil {
		log.Error().Err(err).Msg("")
		return "", err
	}

	if response.Items().Len() != 1 {
		log.Error().Int("orgIDs length", response.Items().Len()).Msg("More than one organization for the given orgID")
		return "", fmt.Errorf("More than one organization for the given orgID (%d)", orgID)
	}

	internalID, ok := response.Items().Get(0).GetID()
	if !ok {
		log.Error().Msgf("Organization %d doesn't have proper internal ID", orgID)
		return "", fmt.Errorf("Organization %d doesn't have proper internal ID", orgID)
	}

	return internalID, nil
}
