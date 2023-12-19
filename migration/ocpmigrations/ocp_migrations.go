package ocpmigrations

import "github.com/RedHatInsights/insights-results-aggregator/migration"

const (
	ruleErrorKeyTable                   = "rule_error_key"
	clusterRuleUserFeedbackTable        = "cluster_rule_user_feedback"
	clusterReportTable                  = "report"
	clusterRuleToggleTable              = "cluster_rule_toggle"
	clusterUserRuleDisableFeedbackTable = "cluster_user_rule_disable_feedback"
	alterTableDropColumnQuery           = "ALTER TABLE %v DROP COLUMN IF EXISTS %v"
	alterTableAddVarcharColumn          = "ALTER TABLE %v ADD COLUMN %v VARCHAR NOT NULL DEFAULT '-1'"
	alterTableDropPK                    = "ALTER TABLE %v DROP CONSTRAINT IF EXISTS %v_pkey"
	alterTableAddPK                     = "ALTER TABLE %v ADD CONSTRAINT %v_pkey PRIMARY KEY %v"
	userIDColumn                        = "user_id"
)

// UsableOCPMigrations contains all OCP recommendation related migrations
var UsableOCPMigrations = []migration.Migration{
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
	mig0015ModifyFeedbackTables,
	mig0016AddRecommendationsTable,
	mig0017AddSystemWideRuleDisableTable,
	mig0018AddRatingsTable,
	mig0019ModifyRecommendationTable,
	mig0020ModifyAdvisorRatingsTable,
	mig0021AddGatheredAtToReport,
	mig0022CleanupEnableDisableTables,
	mig0023AddReportInfoTable,
	mig0024AddTimestampToRuleHit,
	mig0025AddImpactedToRecommendation,
	mig0026AddAndPopulateOrgIDColumns,
	mig0027CleanupInvalidRowsMissingOrgID,
	mig0028AlterRuleDisablePKAndIndex,
	mig0029DropClusterRuleToggleUserIDColumn,
	mig0030DropRuleDisableUserIDColumn,
	mig0031AlterConstraintDropUserAdvisorRatings,
}
