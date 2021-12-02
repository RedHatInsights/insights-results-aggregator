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

package storage

const (
	// key for organization ID used in structured log messages
	organizationKey = "organization"
	// key for cluster ID used in structured log messages
	clusterKey = "cluster"
	// Key for number of issues found for a cluster
	issuesCountKey = "issues found"
	// key for recommendations selectors used in structured log messages
	selectorsKey = "selectors"
	// key for recommendations' creation time
	createdAtKey = "created_at"
	// closeStatementError error string
	closeStatementError = "Unable to close statement"
	// inClauseError when constructing IN clause fails
	inClauseError = "error constructing WHERE IN clause"
	// recommendationTimestampFormat represents the datetime format of the created_at of recommendation table
	recommendationTimestampFormat = "2006-01-02 15:04:05+00:00"
)
