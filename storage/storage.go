/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
*/

// Package storage contains an implementation of interface between Go code and
// (almost any) SQL database like PostgreSQL, SQLite, or MariaDB. An implementation
// named DBStorage is constructed via New function and it is mandatory to call Close
// for any opened connection to database. The storage might be initialized by Init
// method if database schema is empty.
//
// It is possible to configure connection to selected database by using Configuration
// structure. Currently that structure contains two configurable parameter:
//
// Driver - a SQL driver, like "sqlite3", "pq" etc.
// DataSource - specification of data source. The content of this parameter depends on the database used.
package storage

import (
	"database/sql"
	sql_driver "database/sql/driver"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/lib/pq"
	_ "github.com/lib/pq" // PostgreSQL database driver
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3" // SQLite database driver
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/content"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Storage represents an interface to almost any database or storage system
type Storage interface {
	Init() error
	Close() error
	ListOfOrgs() ([]types.OrgID, error)
	ListOfClustersForOrg(orgID types.OrgID) ([]types.ClusterName, error)
	ReadReportForCluster(orgID types.OrgID, clusterName types.ClusterName) (types.ClusterReport, types.Timestamp, error)
	ReadReportForClusterByClusterName(clusterName types.ClusterName) (types.ClusterReport, types.Timestamp, error)
	GetLatestKafkaOffset() (types.KafkaOffset, error)
	WriteReportForCluster(
		orgID types.OrgID,
		clusterName types.ClusterName,
		report types.ClusterReport,
		collectedAtTime time.Time,
		kafkaOffset types.KafkaOffset,
	) error
	ReportsCount() (int, error)
	VoteOnRule(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
		userVote UserVote,
	) error
	AddOrUpdateFeedbackOnRule(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
		message string,
	) error
	GetUserFeedbackOnRule(
		clusterID types.ClusterName, ruleID types.RuleID, userID types.UserID,
	) (*UserFeedbackOnRule, error)
	GetContentForRules(rules types.ReportRules) ([]types.RuleContentResponse, error)
	DeleteReportsForOrg(orgID types.OrgID) error
	DeleteReportsForCluster(clusterName types.ClusterName) error
	ToggleRuleForCluster(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
		ruleToggle RuleToggle,
	) error
	ListDisabledRulesForCluster(
		clusterID types.ClusterName,
		userID types.UserID,
	) ([]types.DisabledRuleResponse, error)
	GetFromClusterRuleToggle(
		types.ClusterName,
		types.RuleID,
		types.UserID,
	) (*ClusterRuleToggle, error)
	DeleteFromRuleClusterToggle(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
	) error
	LoadRuleContent(contentDir content.RuleContentDirectory) error
	GetRuleByID(ruleID types.RuleID) (*types.Rule, error)
	GetOrgIDByClusterID(cluster types.ClusterName) (types.OrgID, error)
	CreateRule(ruleData types.Rule) error
	DeleteRule(ruleID types.RuleID) error
	CreateRuleErrorKey(ruleErrorKey types.RuleErrorKey) error
	DeleteRuleErrorKey(ruleID types.RuleID, errorKey types.ErrorKey) error
	WriteConsumerError(msg *sarama.ConsumerMessage, consumerErr error) error
}

// DBStorage is an implementation of Storage interface that use selected SQL like database
// like SQLite, PostgreSQL, MariaDB, RDS etc. That implementation is based on the standard
// sql package. It is possible to configure connection via Configuration structure.
// SQLQueriesLog is log for sql queries, default is nil which means nothing is logged
type DBStorage struct {
	connection   *sql.DB
	dbDriverType types.DBDriver
	// clusterLastCheckedDict is a dictionary of timestamps when the clusters were last checked.
	clustersLastChecked map[types.ClusterName]time.Time
}

// New function creates and initializes a new instance of Storage interface
func New(configuration Configuration) (*DBStorage, error) {
	driverType, driverName, dataSource, err := initAndGetDriver(configuration)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf(
		"Making connection to data storage, driver=%s datasource=%s",
		driverName, dataSource,
	)

	connection, err := sql.Open(driverName, dataSource)
	if err != nil {
		log.Error().Err(err).Msg("Can not connect to data storage")
		return nil, err
	}

	return NewFromConnection(connection, driverType), nil
}

// NewFromConnection function creates and initializes a new instance of Storage interface from prepared connection
func NewFromConnection(connection *sql.DB, dbDriverType types.DBDriver) *DBStorage {
	return &DBStorage{
		connection:          connection,
		dbDriverType:        dbDriverType,
		clustersLastChecked: map[types.ClusterName]time.Time{},
	}
}

// initAndGetDriver initializes driver(with logs if logSQLQueries is true),
// checks if it's supported and returns driver type, driver name, dataSource and error
func initAndGetDriver(configuration Configuration) (driverType types.DBDriver, driverName string, dataSource string, err error) {
	var driver sql_driver.Driver
	driverName = configuration.Driver

	switch driverName {
	case "sqlite3":
		driverType = types.DBDriverSQLite3
		driver = &sqlite3.SQLiteDriver{}
		dataSource = configuration.SQLiteDataSource
	case "postgres":
		driverType = types.DBDriverPostgres
		driver = &pq.Driver{}
		dataSource = fmt.Sprintf(
			"postgresql://%v:%v@%v:%v/%v?%v",
			configuration.PGUsername,
			configuration.PGPassword,
			configuration.PGHost,
			configuration.PGPort,
			configuration.PGDBName,
			configuration.PGParams,
		)
	default:
		err = fmt.Errorf("driver %v is not supported", driverName)
		return
	}

	if configuration.LogSQLQueries {
		logger := zerolog.New(os.Stdout).With().Str("type", "SQL").Logger()
		driverName = InitSQLDriverWithLogs(driver, driverName, &logger)
	}

	return
}

// MigrateToLatest migrates the database to the latest available
// migration version. This must be done before an Init() call.
func (storage DBStorage) MigrateToLatest() error {
	if err := migration.InitInfoTable(storage.connection); err != nil {
		return err
	}

	return migration.SetDBVersion(storage.connection, storage.dbDriverType, migration.GetMaxVersion())
}

// Init performs all database initialization
// tasks necessary for further service operation.
func (storage DBStorage) Init() error {
	// Read clusterName:LastChecked dictionary from DB.
	rows, err := storage.connection.Query("SELECT cluster, last_checked_at FROM report")
	if err != nil {
		return err
	}

	for rows.Next() {
		var (
			clusterName types.ClusterName
			lastChecked time.Time
		)

		if err := rows.Scan(&clusterName, &lastChecked); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				log.Error().Err(closeErr).Msg("Unable to close the DB rows handle")
			}
			return err
		}

		storage.clustersLastChecked[clusterName] = lastChecked
	}

	// Not using defer to close the rows here to:
	// - make errcheck happy (it doesn't like ignoring returned errors),
	// - return a possible error returned by the Close method.
	return rows.Close()
}

// Close method closes the connection to database. Needs to be called at the end of application lifecycle.
func (storage DBStorage) Close() error {
	log.Info().Msg("Closing connection to data storage")
	if storage.connection != nil {
		err := storage.connection.Close()
		if err != nil {
			log.Error().Err(err).Msg("Can not close connection to data storage")
			return err
		}
	}
	return nil
}

// Report represents one (latest) cluster report.
//     Org: organization ID
//     Name: cluster GUID in the following format:
//         c8590f31-e97e-4b85-b506-c45ce1911a12
type Report struct {
	Org        types.OrgID         `json:"org"`
	Name       types.ClusterName   `json:"cluster"`
	Report     types.ClusterReport `json:"report"`
	ReportedAt types.Timestamp     `json:"reported_at"`
}

func closeRows(rows *sql.Rows) {
	_ = rows.Close()
}

// ListOfOrgs reads list of all organizations that have at least one cluster report
func (storage DBStorage) ListOfOrgs() ([]types.OrgID, error) {
	orgs := make([]types.OrgID, 0)

	rows, err := storage.connection.Query("SELECT DISTINCT org_id FROM report ORDER BY org_id")
	err = types.ConvertDBError(err)
	if err != nil {
		return orgs, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var orgID types.OrgID

		err = rows.Scan(&orgID)
		if err == nil {
			orgs = append(orgs, orgID)
		} else {
			log.Error().Err(err).Msg("ListOfOrgID")
		}
	}
	return orgs, nil
}

// ListOfClustersForOrg reads list of all clusters fro given organization
func (storage DBStorage) ListOfClustersForOrg(orgID types.OrgID) ([]types.ClusterName, error) {
	clusters := make([]types.ClusterName, 0)

	rows, err := storage.connection.Query("SELECT cluster FROM report WHERE org_id = $1 ORDER BY cluster", orgID)
	err = types.ConvertDBError(err)
	if err != nil {
		return clusters, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var clusterName string

		err = rows.Scan(&clusterName)
		if err == nil {
			clusters = append(clusters, types.ClusterName(clusterName))
		} else {
			log.Error().Err(err).Msg("ListOfClustersForOrg")
		}
	}
	return clusters, nil
}

// GetOrgIDByClusterID reads OrgID for specified cluster
func (storage DBStorage) GetOrgIDByClusterID(cluster types.ClusterName) (types.OrgID, error) {
	row := storage.connection.QueryRow("SELECT org_id FROM report WHERE cluster = $1 ORDER BY org_id", cluster)

	var orgID uint64
	err := row.Scan(&orgID)
	if err != nil {
		log.Error().Err(err).Msg("GetOrgIDByClusterID")
		return 0, err
	}
	return types.OrgID(orgID), nil
}

// ReadReportForCluster reads result (health status) for selected cluster for given organization
func (storage DBStorage) ReadReportForCluster(
	orgID types.OrgID, clusterName types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	var report string
	var lastChecked time.Time

	err := storage.connection.QueryRow(
		"SELECT report, last_checked_at FROM report WHERE org_id = $1 AND cluster = $2", orgID, clusterName,
	).Scan(&report, &lastChecked)
	err = types.ConvertDBError(err)

	switch {
	case err == sql.ErrNoRows:
		return "", "", &types.ItemNotFoundError{
			ItemID: fmt.Sprintf("%v/%v", orgID, clusterName),
		}
	case err != nil:
		return "", "", err
	}

	return types.ClusterReport(report), types.Timestamp(lastChecked.UTC().Format(time.RFC3339)), nil
}

// ReadReportForClusterByClusterName reads result (health status) for selected cluster for given organization
func (storage DBStorage) ReadReportForClusterByClusterName(
	clusterName types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	var report string
	var lastChecked time.Time

	err := storage.connection.QueryRow(
		"SELECT report, last_checked_at FROM report WHERE cluster = $1", clusterName,
	).Scan(&report, &lastChecked)

	switch {
	case err == sql.ErrNoRows:
		return "", "", &types.ItemNotFoundError{
			ItemID: fmt.Sprintf("%v", clusterName),
		}
	case err != nil:
		return "", "", err
	}

	return types.ClusterReport(report), types.Timestamp(lastChecked.UTC().Format(time.RFC3339)), nil
}

// GetLatestKafkaOffset returns latest kafka offset from report table
func (storage DBStorage) GetLatestKafkaOffset() (types.KafkaOffset, error) {
	var offset types.KafkaOffset
	err := storage.connection.QueryRow(`SELECT COALESCE(MAX(kafka_offset), 0) FROM report`).Scan(&offset)
	return offset, err
}

// constructWhereClause constructs a dynamic WHERE .. IN clause
// If the rules list is empty, returns NULL to have a syntactically correct WHERE NULL, selecting nothing
func constructWhereClauseForContent(reportRules types.ReportRules) string {
	if len(reportRules.HitRules) == 0 {
		return "NULL" // WHERE NULL
	}
	statement := "(error_key, rule_module) IN (%v)"
	var values string

	for i, rule := range reportRules.HitRules {
		singleVal := ""
		module := strings.TrimSuffix(rule.Module, ".report") // remove trailing .report from module name

		if i == 0 {
			singleVal = fmt.Sprintf(`VALUES ('%v', '%v')`, rule.ErrorKey, module)
		} else {
			singleVal = fmt.Sprintf(`, ('%v', '%v')`, rule.ErrorKey, module)
		}
		values = values + singleVal
	}
	statement = fmt.Sprintf(statement, values)
	return statement
}

func getExtraDataFromReportRules(rules []types.RuleContentResponse, reportRules types.ReportRules) []types.RuleContentResponse {
	if len(reportRules.HitRules) == 0 {
		return rules
	}

	for i, ruleContent := range rules {
		module := ruleContent.RuleModule

		for _, hitRule := range reportRules.HitRules {
			moduleOnReport := strings.TrimSuffix(hitRule.Module, ".report")
			if module == moduleOnReport {
				rules[i].TemplateData = hitRule.Details
			}
		}
	}
	return rules
}

// GetContentForRules retrieves content for rules that were hit in the report
func (storage DBStorage) GetContentForRules(reportRules types.ReportRules) ([]types.RuleContentResponse, error) {
	rules := make([]types.RuleContentResponse, 0)

	query := `
	SELECT
		rek.error_key,
		rek.rule_module,
		rek.description,
		rek.generic,
		r.reason,
		r.resolution,
		rek.publish_date,
		rek.impact,
		rek.likelihood,
		COALESCE(crt.disabled, 0) as disabled
	FROM
		rule r
	INNER JOIN
		rule_error_key rek ON r.module = rek.rule_module
	LEFT JOIN
		cluster_rule_toggle crt ON rek.rule_module = crt.rule_id
	WHERE %v
	ORDER BY
		disabled ASC
	`

	whereInStatement := constructWhereClauseForContent(reportRules)
	query = fmt.Sprintf(query, whereInStatement)

	rows, err := storage.connection.Query(query)

	if err != nil {
		return rules, err
	}
	defer closeRows(rows)

	for rows.Next() {
		var rule types.RuleContentResponse
		var impact, likelihood int

		err = rows.Scan(
			&rule.ErrorKey,
			&rule.RuleModule,
			&rule.Description,
			&rule.Generic,
			&rule.Reason,
			&rule.Resolution,
			&rule.CreatedAt,
			&impact,
			&likelihood,
			&rule.Disabled,
		)
		if err != nil {
			log.Error().Err(err).Msg("SQL error while retrieving content for rule")
			continue
		}

		rule.TotalRisk = (impact + likelihood) / 2

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("SQL rows error while retrieving content for rules")
		return rules, err
	}

	rules = getExtraDataFromReportRules(rules, reportRules)

	return rules, nil
}

func (storage DBStorage) getReportUpsertQuery() (string, error) {
	switch storage.dbDriverType {
	case types.DBDriverSQLite3:
		return `INSERT OR REPLACE INTO report(org_id, cluster, report, reported_at, last_checked_at, kafka_offset)
		 VALUES ($1, $2, $3, $4, $5, $6)`, nil
	case types.DBDriverPostgres:
		return `INSERT INTO report(org_id, cluster, report, reported_at, last_checked_at, kafka_offset)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (org_id, cluster)
		 DO UPDATE SET report = $3, reported_at = $4, last_checked_at = $5, kafka_offset = $6`, nil
	default:
		return "", fmt.Errorf("writing report with DB %v is not supported", storage.dbDriverType)
	}
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage DBStorage) WriteReportForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	lastCheckedTime time.Time,
	kafkaOffset types.KafkaOffset,
) error {
	// Skip writing the report if it isn't newer than a report
	// that is already in the database for the same cluster.
	if oldLastChecked, exists := storage.clustersLastChecked[clusterName]; exists && !lastCheckedTime.After(oldLastChecked) {
		return types.ErrOldReport
	}

	// Get the UPSERT query for writing a report into the database.
	upsertQuery, err := storage.getReportUpsertQuery()
	if err != nil {
		return err
	}

	// Begin a new transaction.
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	// Check if there is a more recent report for the cluster already in the database.
	rows, err := tx.Query(
		`SELECT last_checked_at FROM report WHERE org_id = $1 AND cluster = $2 AND last_checked_at > $3`,
		orgID, clusterName, lastCheckedTime)
	err = types.ConvertDBError(err)
	if err != nil {
		log.Error().Err(err).Msg("Unable to look up the most recent report in database")
		_ = tx.Rollback()
		return err
	}
	defer closeRows(rows)

	// If there is one, print a warning and discard the report (don't update it).
	if rows.Next() {
		log.Warn().Msgf("Database already contains report for organization %d and cluster name %s more recent than %v",
			orgID, clusterName, lastCheckedTime)
		_ = tx.Rollback()
		return nil
	}

	// Perform the report upsert.
	reportedAtTime := time.Now()
	_, err = tx.Exec(upsertQuery, orgID, clusterName, report, reportedAtTime, lastCheckedTime, kafkaOffset)
	if err != nil {
		log.Err(err).Msgf("Unable to upsert the cluster report (org: %v, cluster: %v)", orgID, clusterName)
		_ = tx.Rollback()
		return err
	}

	storage.clustersLastChecked[clusterName] = lastCheckedTime
	metrics.WrittenReports.Inc()
	return tx.Commit()
}

// ReportsCount reads number of all records stored in database
func (storage DBStorage) ReportsCount() (int, error) {
	count := -1
	err := storage.connection.QueryRow("SELECT count(*) FROM report").Scan(&count)
	err = types.ConvertDBError(err)

	return count, err
}

// DeleteReportsForOrg deletes all reports related to the specified organization from the storage.
func (storage DBStorage) DeleteReportsForOrg(orgID types.OrgID) error {
	_, err := storage.connection.Exec("DELETE FROM report WHERE org_id = $1", orgID)
	return err
}

// DeleteReportsForCluster deletes all reports related to the specified cluster from the storage.
func (storage DBStorage) DeleteReportsForCluster(clusterName types.ClusterName) error {
	_, err := storage.connection.Exec("DELETE FROM report WHERE cluster = $1", clusterName)
	return err
}

// loadRuleErrorKeyContent inserts the error key contents of all available rules into the database.
func loadRuleErrorKeyContent(tx *sql.Tx, ruleConfig content.GlobalRuleConfig, ruleModuleName string, errorKeys map[string]content.RuleErrorKeyContent) error {
	for errName, errProperties := range errorKeys {
		var errIsActiveStatus bool
		switch strings.ToLower(errProperties.Metadata.Status) {
		case "active":
			errIsActiveStatus = true
		case "inactive":
			errIsActiveStatus = false
		default:
			_ = tx.Rollback()
			return fmt.Errorf("invalid rule error key status: '%s'", errProperties.Metadata.Status)
		}

		_, err := tx.Exec(`INSERT INTO rule_error_key(error_key, rule_module, condition,
				description, impact, likelihood, publish_date, active, generic)
				VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			errName,
			ruleModuleName,
			errProperties.Metadata.Condition,
			errProperties.Metadata.Description,
			// This will panic if the impact string does not exist as a key in the impact
			// dictionary, which is correct because we cannot continue if that happens.
			ruleConfig.Impact[errProperties.Metadata.Impact],
			errProperties.Metadata.Likelihood,
			errProperties.Metadata.PublishDate,
			errIsActiveStatus,
			errProperties.Generic)

		if err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return nil
}

// LoadRuleContent loads the parsed rule content into the database.
func (storage DBStorage) LoadRuleContent(contentDir content.RuleContentDirectory) error {
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	// SQLite doesn't support `TRUNCATE`, so it's necessary to use `DELETE` and then `VACUUM`.
	if _, err := tx.Exec("DELETE FROM rule_error_key; DELETE FROM rule;"); err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, rule := range contentDir.Rules {
		_, err := tx.Exec(`
				INSERT INTO rule(module, "name", summary, reason, resolution, more_info)
				VALUES($1, $2, $3, $4, $5, $6)`,
			rule.Plugin.PythonModule,
			rule.Plugin.Name,
			rule.Summary,
			rule.Reason,
			rule.Resolution,
			rule.MoreInfo,
		)

		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if err := loadRuleErrorKeyContent(tx, contentDir.Config, rule.Plugin.PythonModule, rule.ErrorKeys); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// GetRuleByID gets a rule by ID
func (storage DBStorage) GetRuleByID(ruleID types.RuleID) (*types.Rule, error) {
	var rule types.Rule

	err := storage.connection.QueryRow(`
		SELECT
			"module",
			"name",
			"summary",
			"reason",
			"resolution",
			"more_info"
		FROM rule WHERE "module" = $1`, ruleID,
	).Scan(
		&rule.Module,
		&rule.Name,
		&rule.Summary,
		&rule.Reason,
		&rule.Resolution,
		&rule.MoreInfo,
	)
	if err == sql.ErrNoRows {
		return nil, &types.ItemNotFoundError{ItemID: ruleID}
	}

	return &rule, err
}

// CreateRule creates rule with provided ruleData in the DB
func (storage DBStorage) CreateRule(ruleData types.Rule) error {
	_, err := storage.connection.Exec(`
		INSERT INTO rule("module", "name", "summary", "reason", "resolution", "more_info")
		VALUES($1, $2, $3, $4, $5, $6)
		ON CONFLICT ("module")
		DO UPDATE SET
			"name" = $2,
			"summary" = $3,
			"reason" = $4,
			"resolution" = $5,
			"more_info" = $6
		;
	`,
		ruleData.Module,
		ruleData.Name,
		ruleData.Summary,
		ruleData.Reason,
		ruleData.Resolution,
		ruleData.MoreInfo,
	)

	return err
}

// DeleteRule deletes rule with provided ruleData in the DB
func (storage DBStorage) DeleteRule(ruleID types.RuleID) error {
	res, err := storage.connection.Exec(`DELETE FROM rule WHERE "module" = $1;`, ruleID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return &types.ItemNotFoundError{ItemID: ruleID}
	}

	return nil
}

// CreateRuleErrorKey creates rule_error_key with provided data in the DB
func (storage DBStorage) CreateRuleErrorKey(ruleErrorKey types.RuleErrorKey) error {
	_, err := storage.connection.Exec(`
		INSERT INTO rule_error_key(
			"error_key",
			"rule_module",
			"condition",
			"description",
			"impact",
			"likelihood",
			"publish_date",
			"active",
			"generic"
		)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT ("error_key", "rule_module")
		DO UPDATE SET
			"condition" = $3,
			"description" = $4,
			"impact" = $5,
			"likelihood" = $6,
			"publish_date" = $7,
			"active" = $8,
			"generic" = $9
		;
	`,
		ruleErrorKey.ErrorKey,
		ruleErrorKey.RuleModule,
		ruleErrorKey.Condition,
		ruleErrorKey.Description,
		ruleErrorKey.Impact,
		ruleErrorKey.Likelihood,
		ruleErrorKey.PublishDate,
		ruleErrorKey.Active,
		ruleErrorKey.Generic,
	)

	return err
}

// DeleteRuleErrorKey creates rule_error_key with provided data in the DB
func (storage DBStorage) DeleteRuleErrorKey(ruleID types.RuleID, errorKey types.ErrorKey) error {
	res, err := storage.connection.Exec(
		`DELETE FROM rule_error_key WHERE "error_key" = $1 AND "rule_module" = $2`,
		errorKey,
		ruleID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return &types.ItemNotFoundError{ItemID: fmt.Sprintf("%v/%v", ruleID, errorKey)}
	}

	return nil
}

// GetConnection returns db connection(useful for testing)
func (storage DBStorage) GetConnection() *sql.DB {
	return storage.connection
}

// WriteConsumerError writes a report about a consumer error into the storage.
func (storage DBStorage) WriteConsumerError(msg *sarama.ConsumerMessage, consumerErr error) error {
	_, err := storage.connection.Exec(`
		INSERT INTO consumer_error (topic, partition, topic_offset, key, produced_at, consumed_at, message, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		msg.Topic, msg.Partition, msg.Offset, msg.Key, msg.Timestamp, time.Now().UTC(), msg.Value, consumerErr.Error())

	return err
}

// GetDBDriverType returns db driver type
func (storage DBStorage) GetDBDriverType() types.DBDriver {
	return storage.dbDriverType
}
