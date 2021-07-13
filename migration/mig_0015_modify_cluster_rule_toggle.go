/*
Copyright © 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package migration

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

/*
	migration15 adds error_key and set as a new primary key
	This make us available to disable rule results instead of whole rules
	for rules with more than one possible error key as result
*/

// mig0015ClusterRuleToggle is a helper for update the cluster_rule_toggle table
var mig0015ClusterRuleToggle = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		var err error

		_, err = tx.Exec(`
			ALTER TABLE cluster_rule_toggle ADD error_key VARCHAR DEFAULT "" NOT NULL;
		`)
		if err != nil {
			return err
		}

		if driver == types.DBDriverPostgres {
			_, err = tx.Exec(`
				ALTER TABLE cluster_rule_toggle DROP CONSTRAINT cluster_rule_toggle_pkey,
					ADD CONSTRAINT cluster_rule_toggle_pkey PRIMARY KEY (cluster_id, rule_id, error_key);
			`)

		} else {
			err = mig0015UpdatePrimaryKeysClusterRuleToggle.StepUp(tx, driver)
		}

		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if driver == types.DBDriverPostgres {
			_, err := tx.Exec(`
				ALTER TABLE cluster_rule_toggle DROP COLUMN error_key;
				ALTER TABLE cluster_rule_toggle DROP CONSTRAINT cluster_rule_toggle_pkey,
				ADD CONSTRAINT cluster_rule_toggle_pkey PRIMARY KEY (cluster_id, rule_id);
			`)
			return err
		}

		return mig0015UpdatePrimaryKeysClusterRuleToggle.StepDown(tx, driver)
	},
}

// mig0015UpdatePrimaryKeysClusterRuleToggle is a helper to update PKs on cluster_rule_toggle table in SQLite
var mig0015UpdatePrimaryKeysClusterRuleToggle = NewUpdateTableMigration(
	clusterRuleToggleTable,
	`
		CREATE TABLE cluster_rule_toggle (
			cluster_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			user_id VARCHAR NULL,
			disabled SMALLINT NOT NULL,
			disabled_at TIMESTAMP NULL,
			enabled_at TIMESTAMP NULL,
			updated_at TIMESTAMP NOT NULL,

			CHECK (disabled >= 0 AND disabled <= 1),
			PRIMARY KEY(cluster_id, rule_id)
		)
	`,
	[]string{"cluster_id", "rule_id", "user_id", "disabled", "disabled_at", "enabled_at", "updated_at"},
	`
		CREATE TABLE cluster_rule_toggle (
			cluster_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			user_id VARCHAR NULL,
			disabled SMALLINT NOT NULL,
			disabled_at TIMESTAMP NULL,
			enabled_at TIMESTAMP NULL,
			updated_at TIMESTAMP NOT NULL,
			error_key VARCHAR NOT NULL,

			CHECK (disabled >= 0 AND disabled <= 1),
			PRIMARY KEY(cluster_id, rule_id, error_key)
		)
	`,
)

// mig0015ClusterRuleUserFeedback is a helper for update the cluster_rule_user_feedback table
var mig0015ClusterRuleUserFeedback = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		var err error
		_, err = tx.Exec(`
			ALTER TABLE cluster_rule_user_feedback ADD error_key VARCHAR DEFAULT "" NOT NULL;
		`)
		if err != nil {
			return err
		}

		if driver == types.DBDriverPostgres {
			_, err = tx.Exec(`
				ALTER TABLE cluster_rule_user_feedback DROP CONSTRAINT cluster_rule_user_feedback_pkey,
					ADD CONSTRAINT cluster_rule_user_feedback_pkey PRIMARY KEY (cluster_id, rule_id, user_id. error_key);
			`)

		} else {
			err = mig0015UpdatePrimaryKeysClusterRuleUserFeedback.StepUp(tx, driver)
		}

		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		return mig0015UpdatePrimaryKeysClusterRuleUserFeedback.StepDown(tx, driver)
	},
}

var mig0015UpdatePrimaryKeysClusterRuleUserFeedback = NewUpdateTableMigration(
	clusterRuleUserFeedbackTable,
	`
		CREATE TABLE cluster_rule_user_feedback (
			cluster_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			user_id VARCHAR NOT NULL,
			message VARCHAR NOT NULL,
			user_vote SMALLINT NOT NULL,
			added_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			
			PRIMARY KEY(cluster_id, rule_id, user_id),
			FOREIGN KEY (cluster_id) REFERENCES report(cluster) ON DELETE CASCADE
		)
	`,
	[]string{"cluster_id", "rule_id", "user_id", "message", "user_vote", "added_at", "updated_at"},
	`
	CREATE TABLE cluster_rule_user_feedback (
		cluster_id VARCHAR NOT NULL,
		rule_id VARCHAR NOT NULL,
		user_id VARCHAR NOT NULL,
		message VARCHAR NOT NULL,
		user_vote SMALLINT NOT NULL,
		added_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		error_key VARCHAR NOT NULL,
		
		PRIMARY KEY(cluster_id, rule_id, user_id, error_key),
		FOREIGN KEY (cluster_id) REFERENCES report(cluster) ON DELETE CASCADE
	)
	`,
)

// migrateClusterRoleToggleData is a helper to update the current data with default values
// It takes the only possible value for error_key on the rules that only has one possible error key
func migrateClusterRoleToggleData(tx *sql.Tx) error {
	updateClusterRuleToggleQuery := `
	UPDATE cluster_rule_toggle SET error_key=$1 WHERE rule_id LIKE $2
	`
	updateClusterRuleUserFeedback := `
	UPDATE cluster_rule_user_feedback SET error_key=$1 WHERE rule_id LIKE $2
	`

	for ruleID, errorKey := range defaultErrorKeysPerRuleID {
		ruleIDWildcard := fmt.Sprintf("%s%%", ruleID)

		log.Info().Str("rule_id", ruleIDWildcard).Str("errorKey", errorKey).Msg("Updating DB data")
		result, err := tx.Exec(updateClusterRuleToggleQuery, errorKey, ruleIDWildcard)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err == nil {
			log.Info().Str("table", "cluster_rule_toggle").Msgf("Rows affected: %d", rowsAffected)
		} else {
			log.Error().Err(err).Msg("Error looking at query result")
		}

		result, err = tx.Exec(updateClusterRuleUserFeedback, errorKey, ruleIDWildcard)
		if err != nil {
			return err
		}

		rowsAffected, err = result.RowsAffected()
		if err == nil {
			log.Info().Str("table", "cluster_rule_user_feedback").Msgf("Rows affected: %d", rowsAffected)
		}
	}

	return nil
}

// mig0015ModifyClusterRuleTables migrates the tables related to user toggle and feedback with error_key
var mig0015ModifyClusterRuleTables = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		if err := mig0015ClusterRuleToggle.StepUp(tx, driver); err != nil {
			return err
		}

		if err := mig0015ClusterRuleUserFeedback.StepUp(tx, driver); err != nil {
			return err
		}

		return migrateClusterRoleToggleData(tx)
	},

	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		if err := mig0015ClusterRuleToggle.StepDown(tx, driver); err != nil {
			return err
		}

		if err := mig0015ClusterRuleUserFeedback.StepDown(tx, driver); err != nil {
			return err
		}

		return nil
	},
}

var defaultErrorKeysPerRuleID map[string]string = map[string]string{
	"ccx_rules_ocp.external.bug_rules.bug_1765280": "BUGZILLA_BUG_1765280",
	"ccx_rules_ocp.external.bug_rules.bug_1766907": "BUGZILLA_BUG_1766907",
	"ccx_rules_ocp.external.bug_rules.bug_1798049": "BUGZILLA_BUG_1798049",
	"ccx_rules_ocp.external.bug_rules.bug_1821905": "BUGZILLA_BUG_1821905",
	"ccx_rules_ocp.external.bug_rules.bug_1832986": "BUGZILLA_BUG_1832986",
	"ccx_rules_ocp.external.bug_rules.bug_1893386": "BUGZILLA_BUG_1893386",

	"ccx_rules_ocp.external.rules.ccxdev_auxiliary_rule":                      "CCXDEV_E2E_TEST_RULE",
	"ccx_rules_ocp.external.rules.check_sap_sdi_observer_pods":                "SAP_SDI_OBSERVER_POD_ERROR",
	"ccx_rules_ocp.external.rules.check_sdi_preload_kernel_modules":           "SDI_PRELOAD_KERNEL_MODULES_ERROR",
	"ccx_rules_ocp.external.rules.cluster_wide_proxy_auth_check":              "AUTH_OPERATOR_PROXY_ERROR",
	"ccx_rules_ocp.external.rules.cmo_container_run_as_non_root":              "CMO_CONTAINER_RUN_AS_NON_ROOT",
	"ccx_rules_ocp.external.rules.container_max_root_partition_size":          "CONTAINER_ROOT_PARTITION_SIZE",
	"ccx_rules_ocp.external.rules.control_plane_replicas":                     "CONTROL_PLANE_NODE_REPLICAS",
	"ccx_rules_ocp.external.rules.empty_prometheus_db_volume":                 "PROMETHEUS_DB_VOLUME_IS_EMPTY",
	"ccx_rules_ocp.external.rules.image_registry_multiple_storage_types":      "IMAGE_REGISTRY_MULTIPLE_STORAGE_TYPES",
	"ccx_rules_ocp.external.rules.lib_bucket_provisioner_check":               "LIB_BUCKET_PROVISIONER_INSTALL_PLANS_ISSUE",
	"ccx_rules_ocp.external.rules.machineconfig_stuck_by_node_taints":         "NODE_HAS_TAINTS_APPLIED",
	"ccx_rules_ocp.external.rules.master_defined_as_machinesets":              "MASTER_DEFINED_AS_MACHINESETS",
	"ccx_rules_ocp.external.rules.node_installer_degraded":                    "NODE_INSTALLER_DEGRADED",
	"ccx_rules_ocp.external.rules.nodes_container_runtime_version_check":      "NODES_CONTAINER_RUNTIME_VERSION",
	"ccx_rules_ocp.external.rules.nodes_kubelet_version_check":                "NODE_KUBELET_VERSION",
	"ccx_rules_ocp.external.rules.nodes_requirements_check":                   "NODES_MINIMUM_REQUIREMENTS_NOT_MET",
	"ccx_rules_ocp.external.rules.openshift_sdn_egress_ip_in_no_hostsubnet":   "OPENSHIFT_SDN_EGRESS_IP_IN_NO_HOSTSUBNET",
	"ccx_rules_ocp.external.rules.operator_unmanaged":                         "OPERATOR_UNMANAGED",
	"ccx_rules_ocp.external.rules.prometheus_backed_by_pvc":                   "PROMETHEUS_BACKED_BY_PVC",
	"ccx_rules_ocp.external.rules.prometheus_rule_evaluation_fail":            "PROMETHEUS_RULE_EVALUATION_FAIL",
	"ccx_rules_ocp.external.rules.same_egress_ip_in_multiple_netnamespaces":   "SAME_EGRESS_IP_IN_MULTIPLE_NETNAMESPACES",
	"ccx_rules_ocp.external.rules.samples_op_failed_image_import_check":       "SAMPLES_FAILED_IMAGE_IMPORT_ERR",
	"ccx_rules_ocp.external.rules.sap_data_intelligence_permissions":          "SAP_DATA_INTELLIGENCE_PERMISSIONS",
	"ccx_rules_ocp.external.rules.subnets_migration_failure_massive_egressip": "SUBNETS_MIGRATION_FAILURE_MASSIVE_EGRESSIP",
	"ccx_rules_ocp.external.rules.tls_handshake_fails_in_azure":               "TLS_HANDSHAKE_FAILS_IN_AZURE",
	"ccx_rules_ocp.external.rules.unsupported_cni_plugin":                     "UNSUPPORTED_CNI_PLUGIN",
	"ccx_rules_ocp.external.rules.vsphere_upi_machine_is_in_phase":            "VSPHERE_UPI_MACHINE_WITH_NO_RUNNING_PHASE",

	// Several ERROR_KEY
	// "ccx_rules_ocp.external.rules.image_registry_storage"
	// "ccx_rules_ocp.external.rules.ocp_version_end_of_life_eus"
	// "ccx_rules_ocp.external.rules.ocp_version_end_of_life"
	// "ccx_rules_ocp.external.rules.openshift_sdn_egress_ip_in_multiple_hostsubnets"
	// "ccx_rules_ocp.external.rules.upgrade_to_ocp47_fails_on_vsphere"
}
