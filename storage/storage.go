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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	_ "github.com/lib/pq"           // PostgreSQL database driver
	_ "github.com/mattn/go-sqlite3" // SQLite database driver

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
	WriteReportForCluster(
		orgID types.OrgID,
		clusterName types.ClusterName,
		report types.ClusterReport,
		collectedAtTime time.Time,
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
}

// DBDriver type for db driver enum
type DBDriver int

const (
	// DBDriverSQLite3 shows that db driver is sqlite
	DBDriverSQLite3 DBDriver = iota
	// DBDriverPostgres shows that db driver is postrgres
	DBDriverPostgres

	// driverNotSupportedMessage is a template for error message displayed
	// in all situations where given DB driver is not supported
	driverNotSupportedMessage = "driver '%v' is not supported"
)

// DBStorage is an implementation of Storage interface that use selected SQL like database
// like SQLite, PostgreSQL, MariaDB, RDS etc. That implementation is based on the standard
// sql package. It is possible to configure connection via Configuration structure.
// SQLQueriesLog is log for sql queries, default is nil which means nothing is logged
type DBStorage struct {
	connection    *sql.DB
	configuration Configuration
	dbDriverType  DBDriver
}

// New function creates and initializes a new instance of Storage interface
func New(configuration Configuration) (*DBStorage, error) {
	driverName := configuration.Driver
	var driverType DBDriver

	switch driverName {
	case "sqlite3":
		driverType = DBDriverSQLite3
	case "postgres":
		driverType = DBDriverPostgres
	default:
		return nil, fmt.Errorf("driver %v is not supported", driverName)
	}

	if configuration.LogSQLQueries {
		// create new logger
		logger := zerolog.New(os.Stdout).With().Str("type", "SQL").Logger()
		var err error
		driverName, err = InitAndGetSQLDriverWithLogs(driverType, &logger)
		if err != nil {
			return nil, err
		}
	}

	dataSource, err := getDataSourceForDriverFromConfig(driverType, configuration)
	if err != nil {
		return nil, fmt.Errorf(driverNotSupportedMessage, configuration.Driver)
	}

	log.Printf("Making connection to data storage, driver=%s datasource=%s", configuration.Driver, dataSource)
	connection, err := sql.Open(driverName, dataSource)

	if err != nil {
		log.Error().Err(err).Msg("Can not connect to data storage")
		return nil, err
	}

	return &DBStorage{
		connection:    connection,
		configuration: configuration,
		dbDriverType:  driverType,
	}, nil
}

func getDataSourceForDriverFromConfig(driverType DBDriver, configuration Configuration) (string, error) {
	switch driverType {
	case DBDriverSQLite3:
		return configuration.SQLiteDataSource, nil
	case DBDriverPostgres:
		return fmt.Sprintf(
			"postgresql://%v:%v@%v:%v/%v?%v",
			configuration.PGUsername,
			configuration.PGPassword,
			configuration.PGHost,
			configuration.PGPort,
			configuration.PGDBName,
			configuration.PGParams,
		), nil
	}

	return "", fmt.Errorf("driver %v is not supported", driverType)
}

// Init method is doing initialization like creating tables in underlying database
func (storage DBStorage) Init() error {
	if err := migration.InitInfoTable(storage.connection); err != nil {
		return err
	}

	return migration.SetDBVersion(storage.connection, migration.GetMaxVersion())
}

// Close method closes the connection to database. Needs to be called at the end of application lifecycle.
func (storage DBStorage) Close() error {
	log.Print("Closing connection to data storage")
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

// ReadReportForCluster reads result (health status) for selected cluster for given organization
func (storage DBStorage) ReadReportForCluster(
	orgID types.OrgID, clusterName types.ClusterName,
) (types.ClusterReport, types.Timestamp, error) {
	var report, lastChecked string

	err := storage.connection.QueryRow(
		"SELECT report, last_checked_at FROM report WHERE org_id = $1 AND cluster = $2", orgID, clusterName,
	).Scan(&report, &lastChecked)

	switch {
	case err == sql.ErrNoRows:
		return "", "", &ItemNotFoundError{
			ItemID: fmt.Sprintf("%v/%v", orgID, clusterName),
		}
	case err != nil:
		return "", "", err
	}

	return types.ClusterReport(report), types.Timestamp(lastChecked), nil
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
		module := strings.TrimRight(rule.Module, ".report") // remove trailing .report from module name

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

// GetContentForRules retrieves content for rules that were hit in the report
func (storage DBStorage) GetContentForRules(reportRules types.ReportRules) ([]types.RuleContentResponse, error) {
	rules := make([]types.RuleContentResponse, 0)

	query := `SELECT error_key, rule_module, description, generic, publish_date,
		impact, likelihood
		FROM rule_error_key
		WHERE %v`

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
			&rule.CreatedAt,
			&impact,
			&likelihood,
		)
		if err != nil {
			log.Error().Err(err).Msg("SQL error while retrieving content for rule")
			continue
		}

		rule.TotalRisk = (impact + likelihood) / 2

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Msg("SQL error while retrieving content for rules")
		return rules, err
	}

	return rules, nil
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage DBStorage) WriteReportForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	lastCheckedTime time.Time,
) (outError error) {
	var query string

	switch storage.dbDriverType {
	case DBDriverSQLite3:
		query = `INSERT OR REPLACE INTO report(org_id, cluster, report, reported_at, last_checked_at)
		 VALUES ($1, $2, $3, $4, $5)`
	case DBDriverPostgres:
		query = `INSERT INTO report(org_id, cluster, report, reported_at, last_checked_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (org_id, cluster)
		 DO UPDATE SET report = $3, reported_at = $4, last_checked_at = $5`
	default:
		return fmt.Errorf("writing report with DB %v is not supported", storage.dbDriverType)
	}

	statement, err := storage.connection.Prepare(query)
	if err != nil {
		return err
	}
	defer func() {
		outError = statement.Close()
	}()

	t := time.Now()

	_, err = statement.Exec(orgID, clusterName, report, t, lastCheckedTime)
	if err != nil {
		log.Print(err)
		return err
	}

	metrics.WrittenReports.Inc()

	return nil
}

// ReportsCount reads number of all records stored in database
func (storage DBStorage) ReportsCount() (int, error) {
	count := -1
	err := storage.connection.QueryRow("SELECT count(*) FROM report").Scan(&count)

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
func loadRuleErrorKeyContent(tx *sql.Tx, ruleModuleName string, errorKeys map[string]content.RuleErrorKeyContent) error {
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
			errProperties.Metadata.Impact,
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

	for _, rule := range contentDir {
		_, err := tx.Exec(`INSERT INTO rule(module, "name", summary, reason, resolution, more_info)
				VALUES($1, $2, $3, $4, $5, $6)`,
			rule.Plugin.PythonModule,
			rule.Plugin.Name,
			rule.Summary,
			rule.Reason,
			rule.Resolution,
			rule.MoreInfo)

		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if err := loadRuleErrorKeyContent(tx, rule.Plugin.PythonModule, rule.ErrorKeys); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if _, err := storage.connection.Exec("VACUUM"); err != nil {
		return err
	}

	return nil
}
