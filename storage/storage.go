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
	"time"

	"github.com/Shopify/sarama"
	"github.com/lib/pq"
	_ "github.com/lib/pq" // PostgreSQL database driver
	"github.com/mattn/go-sqlite3"
	_ "github.com/mattn/go-sqlite3" // SQLite database driver
	"github.com/rs/zerolog/log"

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
		rules []types.ReportItem,
		collectedAtTime time.Time,
		kafkaOffset types.KafkaOffset,
	) error
	ReportsCount() (int, error)
	VoteOnRule(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
		userVote types.UserVote,
		voteMessage string,
	) error
	AddOrUpdateFeedbackOnRule(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
		message string,
	) error
	AddFeedbackOnRuleDisable(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
		message string,
	) error
	GetUserFeedbackOnRule(
		clusterID types.ClusterName, ruleID types.RuleID, userID types.UserID,
	) (*UserFeedbackOnRule, error)
	GetUserFeedbackOnRuleDisable(
		clusterID types.ClusterName, ruleID types.RuleID, userID types.UserID,
	) (*UserFeedbackOnRule, error)
	DeleteReportsForOrg(orgID types.OrgID) error
	DeleteReportsForCluster(clusterName types.ClusterName) error
	ToggleRuleForCluster(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
		ruleToggle RuleToggle,
	) error
	GetFromClusterRuleToggle(
		types.ClusterName,
		types.RuleID,
		types.UserID,
	) (*ClusterRuleToggle, error)
	GetTogglesForRules(
		types.ClusterName,
		[]types.RuleOnReport,
		types.UserID,
	) (map[types.RuleID]bool, error)
	DeleteFromRuleClusterToggle(
		clusterID types.ClusterName,
		ruleID types.RuleID,
		userID types.UserID,
	) error
	GetOrgIDByClusterID(cluster types.ClusterName) (types.OrgID, error)
	WriteConsumerError(msg *sarama.ConsumerMessage, consumerErr error) error
	GetUserFeedbackOnRules(
		clusterID types.ClusterName,
		rulesReport []types.RuleOnReport,
		userID types.UserID,
	) (map[types.RuleID]types.UserVote, error)
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
		driverName = InitSQLDriverWithLogs(driver, driverName)
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
	rows, err := storage.connection.Query("SELECT cluster, last_checked_at FROM report;")
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

	rows, err := storage.connection.Query("SELECT DISTINCT org_id FROM report ORDER BY org_id;")
	err = types.ConvertDBError(err, nil)
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

	rows, err := storage.connection.Query("SELECT cluster FROM report WHERE org_id = $1 ORDER BY cluster;", orgID)
	err = types.ConvertDBError(err, orgID)
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
	row := storage.connection.QueryRow("SELECT org_id FROM report WHERE cluster = $1 ORDER BY org_id;", cluster)

	var orgID uint64
	err := row.Scan(&orgID)
	if err != nil {
		log.Error().Err(err).Msg("GetOrgIDByClusterID")
		return 0, err
	}
	return types.OrgID(orgID), nil
}

// ReadReportForCluster reads result (health status) for selected cluster
func (storage DBStorage) ReadReportForCluster(
	orgID types.OrgID, clusterName types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	var report string
	var lastChecked time.Time

	err := storage.connection.QueryRow(
		"SELECT report, last_checked_at FROM report WHERE org_id = $1 AND cluster = $2;", orgID, clusterName,
	).Scan(&report, &lastChecked)
	err = types.ConvertDBError(err, []interface{}{orgID, clusterName})

	return types.ClusterReport(report), types.Timestamp(lastChecked.UTC().Format(time.RFC3339)), err
}

// ReadReportForClusterByClusterName reads result (health status) for selected cluster for given organization
func (storage DBStorage) ReadReportForClusterByClusterName(
	clusterName types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	var report string
	var lastChecked time.Time

	err := storage.connection.QueryRow(
		"SELECT report, last_checked_at FROM report WHERE cluster = $1;", clusterName,
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
	err := storage.connection.QueryRow("SELECT COALESCE(MAX(kafka_offset), 0) FROM report;").Scan(&offset)
	return offset, err
}

func (storage DBStorage) getReportUpsertQuery() string {
	switch storage.dbDriverType {
	case types.DBDriverSQLite3:
		return `INSERT OR REPLACE INTO report(org_id, cluster, report, reported_at, last_checked_at, kafka_offset)
		 VALUES ($1, $2, $3, $4, $5, $6)`
	case types.DBDriverPostgres:
		return `INSERT INTO report(org_id, cluster, report, reported_at, last_checked_at, kafka_offset)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (org_id, cluster)
		 DO UPDATE SET report = $3, reported_at = $4, last_checked_at = $5, kafka_offset = $6`
	}
	return ""
}

func (storage DBStorage) getRuleUpsertQuery() string {
	switch storage.dbDriverType {
	case types.DBDriverSQLite3:
		return `INSERT OR REPLACE INTO rule_hit(org_id, cluster_id, rule_id, error_key, report)
		 VALUES ($1, $2, $3, $4, $5)`
	case types.DBDriverPostgres:
		return `INSERT INTO rule_hit(org_id, cluster_id, rule_id, error_key, report)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (org_id, cluster, rule_id, error_key)
		 DO UPDATE SET report = $4`
	}
	return ""
}

func (storage DBStorage) updateReport(
	tx *sql.Tx,
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	rules []types.ReportItem,
	lastCheckedTime time.Time,
	kafkaOffset types.KafkaOffset,
) error {
	// Get the UPSERT query for writing a report into the database.
	reportUpsertQuery := storage.getReportUpsertQuery()

	// Get the UPSERT query for writing a rule into the database.
	ruleUpsertQuery := storage.getRuleUpsertQuery()

	deleteQuery := "DELETE FROM rule_hit WHERE org_id = $1 AND cluster_id = $2;"
	_, err := tx.Exec(deleteQuery, orgID, clusterName)
	if err != nil {
		log.Err(err).Msgf("Unable to remove previous cluster reports (org: %v, cluster: %v)", orgID, clusterName)
		_ = tx.Rollback()
		return err
	}

	// Perform the report upsert.
	reportedAtTime := time.Now()

	for _, rule := range rules {
		_, err = tx.Exec(ruleUpsertQuery, orgID, clusterName, rule.Module, rule.ErrorKey, string(rule.TemplateData))
		if err != nil {
			log.Err(err).Msgf("Unable to upsert the cluster report (org: %v, cluster: %v)", orgID, clusterName)
			_ = tx.Rollback()
			return err
		}
	}

	_, err = tx.Exec(reportUpsertQuery, orgID, clusterName, report, reportedAtTime, lastCheckedTime, kafkaOffset)
	if err != nil {
		log.Err(err).Msgf("Unable to upsert the cluster report (org: %v, cluster: %v)", orgID, clusterName)
		_ = tx.Rollback()
		return err
	}
	return nil
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage DBStorage) WriteReportForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	rules []types.ReportItem,
	lastCheckedTime time.Time,
	kafkaOffset types.KafkaOffset,
) error {
	// Skip writing the report if it isn't newer than a report
	// that is already in the database for the same cluster.
	if oldLastChecked, exists := storage.clustersLastChecked[clusterName]; exists && !lastCheckedTime.After(oldLastChecked) {
		return types.ErrOldReport
	}

	if storage.dbDriverType != types.DBDriverSQLite3 && storage.dbDriverType != types.DBDriverPostgres {
		return fmt.Errorf("writing report with DB %v is not supported", storage.dbDriverType)
	}

	// Begin a new transaction.
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	// Check if there is a more recent report for the cluster already in the database.
	rows, err := tx.Query(
		"SELECT last_checked_at FROM report WHERE org_id = $1 AND cluster = $2 AND last_checked_at > $3;",
		orgID, clusterName, lastCheckedTime)
	err = types.ConvertDBError(err, []interface{}{orgID, clusterName})
	if err != nil {
		log.Error().Err(err).Msg("Unable to look up the most recent report in database")
		return err
	}
	defer closeRows(rows)

	// If there is one, print a warning and discard the report (don't update it).
	if rows.Next() {
		log.Warn().Msgf("Database already contains report for organization %d and cluster name %s more recent than %v",
			orgID, clusterName, lastCheckedTime)
		return nil
	}

	err = storage.updateReport(tx, orgID, clusterName, report, rules, lastCheckedTime, kafkaOffset)
	if err != nil {
		return err
	}

	storage.clustersLastChecked[clusterName] = lastCheckedTime
	metrics.WrittenReports.Inc()
	return tx.Commit()
}

// ReportsCount reads number of all records stored in database
func (storage DBStorage) ReportsCount() (int, error) {
	count := -1
	err := storage.connection.QueryRow("SELECT count(*) FROM report;").Scan(&count)
	err = types.ConvertDBError(err, nil)

	return count, err
}

// DeleteReportsForOrg deletes all reports related to the specified organization from the storage.
func (storage DBStorage) DeleteReportsForOrg(orgID types.OrgID) error {
	_, err := storage.connection.Exec("DELETE FROM report WHERE org_id = $1;", orgID)
	return err
}

// DeleteReportsForCluster deletes all reports related to the specified cluster from the storage.
func (storage DBStorage) DeleteReportsForCluster(clusterName types.ClusterName) error {
	_, err := storage.connection.Exec("DELETE FROM report WHERE cluster = $1;", clusterName)
	return err
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
