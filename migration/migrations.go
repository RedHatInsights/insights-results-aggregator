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

package migration

// migrations is a list of migrations that, when applied in their order,
// create the most recent version of the database from scratch.
var migrations = []Migration{
	mig0001CreateReport,
	mig0002CreateRuleContent,
	mig0003CreateClusterRuleUserFeedback,
	mig0004ModifyClusterRuleUserFeedback,
	mig0005CreateConsumerError,
	mig0006AddOnDeleteCascade,
	mig0007CreateClusterRuleToggle,
	mig0008AddOffsetFieldToReportTable,
	mig0009AddIndexOnReportKafkaOffset,
	mig0010AddTagsFieldToRuleErrorKeyTable,
	mig0011RemoveFKAndContentTables,
	mig0012CreateClusterUserRuleDisableFeedback,
	mig0013AddRuleHitTable,
	mig0014ModifyClusterRuleToggle,
	mig0015ModifyClusterRuleTables,
}
