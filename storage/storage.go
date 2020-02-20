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

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
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
}

// DBDriver type for db driver enum
type DBDriver int

const (
	// DBDriverSQLite shows that db driver is sqlite
	DBDriverSQLite DBDriver = iota
	// DBDriverPostgres shows that db driver is postrgres
	DBDriverPostgres
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

	var dataSource string
	switch configuration.Driver {
	case "sqlite3":
		dataSource = configuration.SQLiteDataSource
	case "postgres":

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
		return nil, fmt.Errorf("Driver %v is not supported", configuration.Driver)
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
	case strings.HasPrefix(driverName, "postgres"):
		driverType = DBDriverPostgres
	default:
		return nil, fmt.Errorf("driver %v is not supported", driverName)
	}

	return &DBStorage{
		connection:    connection,
		configuration: configuration,
		dbDriverType:  driverType,
	}, nil
}

// Init method is doing initialization like creating tables in underlying database
func (storage DBStorage) Init() error {
	_, err := storage.connection.Exec(`
		create table IF NOT EXISTS report (
			org_id      integer not null,
			cluster     varchar not null unique,
			report      varchar not null,
			reported_at datetime,
			last_checked_at datetime,
			PRIMARY KEY(org_id, cluster)
		);
	`)
	return err
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

// ListOfOrgs reads list of all organizations that have at least one cluster report
func (storage DBStorage) ListOfOrgs() ([]types.OrgID, error) {
	orgs := []types.OrgID{}

	rows, err := storage.connection.Query("SELECT DISTINCT org_id FROM report ORDER BY org_id")
	if err != nil {
		return orgs, err
	}
	defer rows.Close()

	for rows.Next() {
		var orgID types.OrgID

		err = rows.Scan(&orgID)
		if err == nil {
			orgs = append(orgs, types.OrgID(orgID))
		} else {
			log.Println("error", err)
		}
	}
	return orgs, nil
}

// ListOfClustersForOrg reads list of all clusters fro given organization
func (storage DBStorage) ListOfClustersForOrg(orgID types.OrgID) ([]types.ClusterName, error) {
	clusters := []types.ClusterName{}

	rows, err := storage.connection.Query("SELECT cluster FROM report WHERE org_id = $1 ORDER BY cluster", orgID)
	if err != nil {
		return clusters, err
	}
	defer rows.Close()

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
func (storage DBStorage) ReadReportForCluster(orgID types.OrgID, clusterName types.ClusterName) (types.ClusterReport, error) {
	rows, err := storage.connection.Query(
		"SELECT report FROM report WHERE org_id = $1 AND cluster = $2", orgID, clusterName,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var report string

		err = rows.Scan(&report)
		if err == nil {
			return types.ClusterReport(report), nil
		}
		log.Println("error", err)
		return "", err
	}

	return "", &ItemNotFoundError{
		ItemID: fmt.Sprintf("%v/%v", orgID, clusterName),
	}
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage DBStorage) WriteReportForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	lastCheckedTime time.Time,
) error {
	var query string

	switch storage.dbDriverType {
	case DBDriverSQLite:
		query = `INSERT OR REPLACE INTO report(org_id, cluster, report, reported_at, last_checked_at) 
		 VALUES ($1, $2, $3, $4, $5)`
	default:
		query = `INSERT INTO report(org_id, cluster, report, reported_at, last_checked_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (org_id, cluster) 
		 DO UPDATE SET report = $3, reported_at = $4, last_checked_at = $5`
	}
	statement, err := storage.connection.Prepare(query)
	if err != nil {
		return err
	}
	defer statement.Close()

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
	rows, err := storage.connection.Query("SELECT count(*) FROM report")
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	if rows.Next() {
		var cnt int

		err = rows.Scan(&cnt)
		if err == nil {
			return cnt, nil
		}
		log.Println("error", err)
		return -1, err
	}
	return -1, err
}
