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
limitations under the License.
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
	"log"
	"os"
	"strings"
	"time"

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
	ReadReportForCluster(orgID types.OrgID, clusterName types.ClusterName) (types.ClusterReport, error)
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
}

// DBDriver type for db driver enum
type DBDriver int

const (
	// DBDriverSQLite shows that db driver is sqlite
	DBDriverSQLite DBDriver = iota
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

	if configuration.LogSQLQueries {
		logger := log.New(os.Stdout, "[sql]", log.LstdFlags)
		var err error
		driverName, err = InitAndGetSQLDriverWithLogs(driverName, logger)
		if err != nil {
			return nil, err
		}
	}

	dataSource, err := getDataSourceFromConfig(configuration)
	if err != nil {
		return nil, fmt.Errorf(driverNotSupportedMessage, configuration.Driver)
	}

	log.Printf("Making connection to data storage, driver=%s datasource=%s", configuration.Driver, dataSource)
	connection, err := sql.Open(configuration.Driver, dataSource)

	if err != nil {
		log.Println("Can not connect to data storage", err)
		return nil, err
	}

	var driverType DBDriver

	switch {
	case strings.HasPrefix(driverName, "sqlite"):
		driverType = DBDriverSQLite
	case strings.HasPrefix(driverName, postgresDriverName):
		driverType = DBDriverPostgres
	default:
		return nil, fmt.Errorf(driverNotSupportedMessage, driverName)
	}

	return &DBStorage{
		connection:    connection,
		configuration: configuration,
		dbDriverType:  driverType,
	}, nil
}

func getDataSourceFromConfig(configuration Configuration) (string, error) {
	switch configuration.Driver {
	case sqliteDriverName:
		return configuration.SQLiteDataSource, nil
	case postgresDriverName:
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

	return "", fmt.Errorf("driver %v is not supported", configuration.Driver)
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
	log.Println("Closing connection to data storage")
	if storage.connection != nil {
		err := storage.connection.Close()
		if err != nil {
			log.Fatal("Can not close connection to data storage", err)
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
			log.Println("error", err)
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
			log.Println("error", err)
		}
	}
	return clusters, nil
}

// ReadReportForCluster reads result (health status) for selected cluster for given organization
func (storage DBStorage) ReadReportForCluster(
	orgID types.OrgID, clusterName types.ClusterName,
) (types.ClusterReport, error) {
	var report string
	err := storage.connection.QueryRow(
		"SELECT report FROM report WHERE org_id = $1 AND cluster = $2", orgID, clusterName,
	).Scan(&report)

	switch {
	case err == sql.ErrNoRows:
		return "", &ItemNotFoundError{
			ItemID: fmt.Sprintf("%v/%v", orgID, clusterName),
		}
	case err != nil:
		return "", err
	}

	return types.ClusterReport(report), nil
}

// constructWhereClause constructs a dynamic WHERE .. IN clause
func constructWhereClause(reportRules types.ReportRules) string {
	var statement string

	for i, rule := range reportRules.Rules {
		singleVal := ""
		module := strings.TrimRight(rule.Module, ".report") // remove trailing .report from module name

		if i == 0 {
			singleVal = fmt.Sprintf("VALUES (\"%v\", \"%v\")", rule.ErrorKey, module)
		} else {
			singleVal = fmt.Sprintf(", (%v, %v)", rule.ErrorKey, module)
		}
		statement = statement + singleVal
	}
	return statement
}

// GetContentForRules retrieves content for rules
func (storage DBStorage) GetContentForRules(reportRules types.ReportRules) ([]types.RuleContentResponse, error) {
	rules := []types.RuleContentResponse{}

	whereInStatement := constructWhereClause(reportRules)

	q := `SELECT rek.error_key, rek.rule_module, rek.description, rek.publish_date,
		rek.impact, rek.likelihood
		FROM rule_error_key rek LEFT JOIN rule r ON rek.rule_module = r.module
		WHERE (rek.error_key, rek.rule_module) IN (%v)`

	q = fmt.Sprintf(q, whereInStatement)

	rows, err := storage.connection.Query(q)

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
			&rule.CreatedAt,
			&impact,
			&likelihood,
		)
		if err != nil {
			log.Println("SQL error in retrieving content for rules", err)
			continue
		}

		rule.TotalRisk = ((impact + likelihood) / 2)

		rules = append(rules, rule)
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
	case DBDriverSQLite:
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
