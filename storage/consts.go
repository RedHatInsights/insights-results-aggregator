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
	// key for current organization ID used in structured log messages
	currentOrgIDKey = "currentOrgID"
	// key for original organization ID used in structured log messages
	originalOrgIDKey = "originalOrgID"
	// key for table name used in structured log messages
	tableNameKey = "tableName"
	// key for number of rows affected by an operation
	rowsAffectedKey = "rowsAffected"
	// closeStatementError error string
	closeStatementError = "Unable to close statement"
	// inClauseError when constructing IN clause fails
	inClauseError = "error constructing WHERE IN clause"
)
