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

package consumer

import (
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	// key for topic name used in structured log messages
	topicKey = "topic"
	// key for broker group name used in structured log messages
	groupKey = "group"
	// key for message offset used in structured log messages
	offsetKey = "offset"
	// key for message partition used in structured log messages
	partitionKey = "partition"
	// key for organization ID used in structured log messages
	organizationKey = "organization"
	// key for cluster ID used in structured log messages
	clusterKey = "cluster"
	// key for duration message type used in structured log messages
	durationKey = "duration"
	// key for data schema version message type used in structured log messages
	versionKey = "version"
	// CurrentSchemaVersion represents the currently supported data schema version
	CurrentSchemaVersion = types.SchemaVersion(1)
)
