/*
Copyright © 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
*/

package storage

import (
	"database/sql"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/migration/dvomigrations"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// DVORecommendationsStorage represents an interface to almost any database or storage system
type DVORecommendationsStorage interface {
	Init() error
	Close() error
	GetMigrations() []migration.Migration
	GetDBDriverType() types.DBDriver
	GetConnection() *sql.DB
	GetMaxVersion() migration.Version
	MigrateToLatest() error
}

// DVORecommendationsDBStorage is an implementation of Storage interface that use selected SQL like database
// like PostgreSQL or RDS etc. That implementation is based on the standard
// sql package. It is possible to configure connection via Configuration structure.
// SQLQueriesLog is log for sql queries, default is nil which means nothing is logged
type DVORecommendationsDBStorage struct {
	connection   *sql.DB
	dbDriverType types.DBDriver
}

// NewDVORecommendationsStorage function creates and initializes a new instance of Storage interface
func NewDVORecommendationsStorage(configuration Configuration) (DVORecommendationsStorage, error) {
	switch configuration.Type {
	case types.SQLStorage:
		log.Info().Str("DVO storage type", configuration.Type).Send()
		return newDVOStorage(configuration)
	case types.NoopStorage:
		return newNoopDVOStorage(configuration)
	default:
		// error to be thrown
		err := fmt.Errorf("Unknown storage type '%s'", configuration.Type)
		log.Error().Err(err).Msg("Init failure")
		return nil, err
	}
}

// newNoopDVOStorage function creates and initializes a new instance of Noop storage
func newNoopDVOStorage(_ Configuration) (DVORecommendationsStorage, error) {
	return &NoopDVOStorage{}, nil
}

// newDVOStorage function creates and initializes a new instance of DB storage
func newDVOStorage(configuration Configuration) (DVORecommendationsStorage, error) {
	driverType, driverName, dataSource, err := initAndGetDriver(configuration)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf(
		"Making connection to DVO data storage, driver=%s",
		driverName,
	)

	connection, err := sql.Open(driverName, dataSource)
	if err != nil {
		log.Error().Err(err).Msg("Can not connect to data storage")
		return nil, err
	}

	return NewDVORecommendationsFromConnection(connection, driverType), nil
}

// NewDVORecommendationsFromConnection function creates and initializes a new instance of Storage interface from prepared connection
func NewDVORecommendationsFromConnection(connection *sql.DB, dbDriverType types.DBDriver) *DVORecommendationsDBStorage {
	return &DVORecommendationsDBStorage{
		connection:   connection,
		dbDriverType: dbDriverType,
	}
}

// Init performs all database initialization
// tasks necessary for further service operation.
func (storage DVORecommendationsDBStorage) Init() error {
	return nil
}

// Close method closes the connection to database. Needs to be called at the end of application lifecycle.
func (storage DVORecommendationsDBStorage) Close() error {
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

// GetMigrations returns a list of database migrations related to DVO recommendation tables
func (storage DVORecommendationsDBStorage) GetMigrations() []migration.Migration {
	return dvomigrations.UsableDVOMigrations
}

// GetDBDriverType returns db driver type
func (storage DVORecommendationsDBStorage) GetDBDriverType() types.DBDriver {
	return storage.dbDriverType
}

// GetConnection returns db connection(useful for testing)
func (storage DVORecommendationsDBStorage) GetConnection() *sql.DB {
	return storage.connection
}

// GetMaxVersion returns the highest available migration version.
// The DB version cannot be set to a value higher than this.
// This value is equivalent to the length of the list of available migrations.
func (storage DVORecommendationsDBStorage) GetMaxVersion() migration.Version {
	return migration.Version(len(storage.GetMigrations()))
}

// MigrateToLatest migrates the database to the latest available
// migration version. This must be done before an Init() call.
func (storage DVORecommendationsDBStorage) MigrateToLatest() error {
	if err := migration.InitInfoTable(storage.connection); err != nil {
		return err
	}

	return migration.SetDBVersion(
		storage.connection,
		storage.dbDriverType,
		storage.GetMaxVersion(),
		storage.GetMigrations(),
	)
}
