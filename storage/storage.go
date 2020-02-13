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

package storage

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"           // PostgreSQL database driver
	_ "github.com/mattn/go-sqlite3" // SQLite database driver

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Storage represents an interface to any relational database based on SQL language
type Storage interface {
	Init() error
	Close() error
	ListOfOrgs() ([]types.OrgID, error)
	ListOfClustersForOrg(orgID types.OrgID) ([]types.ClusterName, error)
	ReadReportForCluster(orgID types.OrgID, clusterName types.ClusterName) (types.ClusterReport, error)
	WriteReportForCluster(orgID types.OrgID, clusterName types.ClusterName, report types.ClusterReport) error
	ReportsCount() (int, error)
}

// Impl is an implementation of Storage interface
type Impl struct {
	connection    *sql.DB
	configuration Configuration
}

// New function creates and initializes a new instance of Storage interface
func New(configuration Configuration) (Storage, error) {
	log.Printf("Making connection to data storage, driver=%s datasource=%s", configuration.Driver, configuration.DataSource)
	connection, err := sql.Open(configuration.Driver, configuration.DataSource)

	if err != nil {
		log.Println("Can not connect to data storage", err)
		return nil, err
	}

	return Impl{connection, configuration}, nil
}

// Init method is doing initialization like creating tables in underlying database
func (storage Impl) Init() error {
	_, err := storage.connection.Exec(`
		create table report (
			org_id      integer not null,
			cluster     varchar not null unique,
			report      varchar not null,
			reported_at datetime,
			PRIMARY KEY(org_id, cluster)
		);
	`)
	return err
}

// Close method closes the connection to database. Needs to be called at the end of application lifecycle.
func (storage Impl) Close() error {
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
func (storage Impl) ListOfOrgs() ([]types.OrgID, error) {
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
func (storage Impl) ListOfClustersForOrg(orgID types.OrgID) ([]types.ClusterName, error) {
	clusters := []types.ClusterName{}

	rows, err := storage.connection.Query("SELECT cluster FROM report WHERE org_id = ? ORDER BY cluster", orgID)
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
func (storage Impl) ReadReportForCluster(orgID types.OrgID, clusterName types.ClusterName) (types.ClusterReport, error) {
	rows, err := storage.connection.Query("SELECT report FROM report WHERE org_id = ? AND cluster = ?", orgID, clusterName)
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
	return "", err
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage Impl) WriteReportForCluster(orgID types.OrgID, clusterName types.ClusterName, report types.ClusterReport) error {
	statement, err := storage.connection.Prepare("INSERT OR REPLACE INTO report(org_id, cluster, report, reported_at) VALUES ($1, $2, $3, $4)")
	if err != nil {
		return err
	}
	defer statement.Close()

	t := time.Now()

	_, err = statement.Exec(orgID, clusterName, report, t)
	if err != nil {
		log.Print(err)
		return err
	}
	metrics.WrittenReports.Inc()
	return nil
}

// ReportsCount reads number of all records stored in database
func (storage Impl) ReportsCount() (int, error) {
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
