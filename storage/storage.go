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
	_ "github.com/lib/pq"           // PostgreSQL database driver
	_ "github.com/mattn/go-sqlite3" // SQLite database driver
	"log"
	"time"
)

// OrgID represents organization ID
type OrgID int

// ClusterName represents name of cluster in format c8590f31-e97e-4b85-b506-c45ce1911a12
type ClusterName string

// ClusterReport represents cluster report
type ClusterReport string

// Timestamp represents any timestamp in a form gathered from database
// TODO: need to be improved
type Timestamp string

// Storage represents an interface to any relational database based on SQL language
type Storage interface {
	Close() error
	ListOfOrgs() ([]OrgID, error)
	ListOfClustersForOrg(orgID OrgID) ([]ClusterName, error)
	ReadReportForCluster(orgID OrgID, clusterName ClusterName) (ClusterReport, error)
	WriteReportForCluster(orgID OrgID, clusterName ClusterName, report ClusterReport) error
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
		log.Fatal("Can not connect to data storage", err)
		return nil, err
	}

	return Impl{connection, configuration}, nil
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
	Org        OrgID         `json:"org"`
	Name       ClusterName   `json:"cluster"`
	Report     ClusterReport `json:"report"`
	ReportedAt Timestamp     `json:"reported_at"`
}

// ListOfOrgs reads list of all organizations that have at least one cluster report
func (storage Impl) ListOfOrgs() ([]OrgID, error) {
	orgs := []OrgID{}

	rows, err := storage.connection.Query("SELECT org_id FROM report ORDER BY org_id")
	if err != nil {
		return orgs, err
	}
	defer rows.Close()

	for rows.Next() {
		var orgID OrgID

		err = rows.Scan(&orgID)
		if err == nil {
			orgs = append(orgs, OrgID(orgID))
		} else {
			log.Println("error", err)
		}
	}
	return orgs, nil
}

// ListOfClustersForOrg reads list of all clusters fro given organization
func (storage Impl) ListOfClustersForOrg(orgID OrgID) ([]ClusterName, error) {
	clusters := []ClusterName{}

	rows, err := storage.connection.Query("SELECT cluster FROM report WHERE org_id = ? ORDER BY cluster", orgID)
	if err != nil {
		return clusters, err
	}
	defer rows.Close()

	for rows.Next() {
		var clusterName string

		err = rows.Scan(&clusterName)
		if err == nil {
			clusters = append(clusters, ClusterName(clusterName))
		} else {
			log.Println("error", err)
		}
	}
	return clusters, nil
}

// ReadReportForCluster reads result (health status) for selected cluster for given organization
func (storage Impl) ReadReportForCluster(orgID OrgID, clusterName ClusterName) (ClusterReport, error) {
	rows, err := storage.connection.Query("SELECT report FROM report WHERE org_id = ? AND cluster = ?", orgID, clusterName)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	if rows.Next() {
		var report string

		err = rows.Scan(&report)
		if err == nil {
			return ClusterReport(report), nil
		}
		log.Println("error", err)
		return "", err
	}
	return "", err
}

// WriteReportForCluster writes result (health status) for selected cluster for given organization
func (storage Impl) WriteReportForCluster(orgID OrgID, clusterName ClusterName, report ClusterReport) error {
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
	return nil
}
