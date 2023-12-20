/*
Copyright © 2021, 2022, 2023 Red Hat, Inc.

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

// Entry point to the insights results aggregator service.
//
// The service contains consumer (usually Kafka consumer) that consumes
// messages from given source, processes those messages and stores them
// in configured data store. It also starts REST API servers with
// endpoints that expose several types of information: list of organizations,
// list of clusters for given organization, and cluster health.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/RedHatInsights/insights-operator-utils/logger"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Exit codes
const (
	// ExitStatusOK means that the tool finished with success
	ExitStatusOK = iota
	// ExitStatusError is a general error code
	ExitStatusError
	// ExitStatusPrepareDbError is returned when the DB preparation (including rule content loading) fails
	ExitStatusPrepareDbError
	// ExitStatusConsumerError is returned in case of any consumer-related error
	ExitStatusConsumerError
	// ExitStatusServerError is returned in case of any REST API server-related error
	ExitStatusServerError
	// ExitStatusMigrationError is returned in case of an error while attempting to perform DB migrations
	ExitStatusMigrationError
)

// Messages
const (
	databasePreparationMessage = "database preparation exited with error code %v"
)

// Other constants
const (
	defaultConfigFilename = "config"
	typeStr               = "type"
)

var (
	// BuildVersion contains the major.minor version of the CLI client.
	// It is set up during build process.
	BuildVersion = "*not set*"

	// BuildTime contains timestamp when the CLI client has been built.
	// It is set up during build process.
	BuildTime = "*not set*"

	// BuildBranch contains Git branch used to build this application.
	// It is set up during build process.
	BuildBranch = "*not set*"

	// BuildCommit contains Git commit used to build this application.
	// It is set up during build process.
	BuildCommit = "*not set*"

	// UtilsVersion contains currently used version of
	// github.com/RedHatInsights/insights-operator-utils package
	UtilsVersion = "*not set*"

	// autoMigrate determines if the prepareDB function upgrades
	// the database to the latest migration version. This is necessary
	// for unit tests that work with an empty DB.
	autoMigrate = false
)

// fillInInfoParams function fills-in additional info used by /info endpoint
// handler
func fillInInfoParams(params map[string]string) {
	params["BuildVersion"] = BuildVersion
	params["BuildTime"] = BuildTime
	params["BuildBranch"] = BuildBranch
	params["BuildCommit"] = BuildCommit
	params["UtilsVersion"] = UtilsVersion
}

// createStorage function initializes connection to preconfigured storage,
// usually PostgreSQL or AWS RDS.
func createStorage() (storage.OCPRecommendationsStorage, storage.DVORecommendationsStorage, error) {
	ocpStorageCfg := conf.GetOCPRecommendationsStorageConfiguration()
	// Redis configuration needs to be present in ocpStorageCfg, as the connection is created in the same function
	ocpStorageCfg.RedisConfiguration = conf.GetRedisConfiguration()

	dvoStorageCfg := conf.GetDVORecommendationsStorageConfiguration()

	var ocpStorage storage.OCPRecommendationsStorage
	var dvoStorage storage.DVORecommendationsStorage
	var err error

	// try to initialize connection to storage
	backend := conf.GetStorageBackendConfiguration().Use
	switch backend {
	case types.OCPRecommendationsStorage:
		ocpStorage, err = storage.NewOCPRecommendationsStorage(ocpStorageCfg)
		if err != nil {
			log.Error().Err(err).Msg("storage.NewOCPRecommendationsStorage")
			return nil, nil, err
		}
	case types.DVORecommendationsStorage:
		dvoStorage, err = storage.NewDVORecommendationsStorage(dvoStorageCfg)
		if err != nil {
			log.Error().Err(err).Msg("storage.NewDVORecommendationsStorage")
			return nil, nil, err
		}
	default:
		return nil, nil, fmt.Errorf("Unknown storage backend %s", backend)
	}

	return ocpStorage, dvoStorage, nil
}

// closeStorage function closes specified DBStorage with proper error checking
// whether the close operation was successful or not.
func closeStorage(storage storage.Storage) {
	err := storage.Close()
	if err != nil {
		// TODO: error state might be returned from this function
		log.Error().Err(err).Msg("Error during closing storage connection")
	}
}

// prepareDBMigrations function checks the actual database version and when
// autoMigrate is set performs migration to the latest schema version
// available.
func prepareDBMigrations(dbStorage storage.Storage) int {
	driverType := dbStorage.GetDBDriverType()
	if driverType != types.DBDriverPostgres {
		log.Info().Msg("Skipping migration for non-SQL database type")
		return ExitStatusOK
	}
	// This is only used by some unit tests.
	if autoMigrate {
		if err := dbStorage.MigrateToLatest(); err != nil {
			log.Error().Err(err).Msg("unable to migrate DB to latest version")
			return ExitStatusPrepareDbError
		}
	} else {
		currentVersion, err := migration.GetDBVersion(dbStorage.GetConnection())
		if err != nil {
			log.Error().Err(err).Msg("unable to check DB migration version")
			return ExitStatusPrepareDbError
		}

		maxVersion := dbStorage.GetMaxVersion()
		if currentVersion != maxVersion {
			log.Error().Msgf("old DB migration version (current: %d, latest: %d)", currentVersion, maxVersion)
			return ExitStatusPrepareDbError
		}
	}

	return ExitStatusOK
}

// prepareDB function opens a connection to database and loads all available
// rule content into it.
func prepareDB() int {
	// TODO: when aggregator supports both storages at once, update the code below
	// task to support both storages at once: https://issues.redhat.com/browse/CCXDEV-12316

	ocpRecommendationsStorage, _, err := createStorage()

	if err != nil {
		log.Error().Err(err).Msg("Error creating storage")
		return ExitStatusPrepareDbError
	}
	defer closeStorage(ocpRecommendationsStorage)

	// Ensure that the DB is at the latest migration version.
	if exitCode := prepareDBMigrations(ocpRecommendationsStorage); exitCode != ExitStatusOK {
		return exitCode
	}

	// Initialize the database.
	err = ocpRecommendationsStorage.Init()
	if err != nil {
		log.Error().Err(err).Msg("DB initialization error")
		return ExitStatusPrepareDbError
	}

	// temporarily print some information from DB because of limited access to DB
	ocpRecommendationsStorage.PrintRuleDisableDebugInfo()

	return ExitStatusOK
}

// startService function starts service and returns error code in case the
// service can't be started properly. If service is terminated correctly,
// ExitStatusOK is returned instead.
func startService() int {
	metricsCfg := conf.GetMetricsConfiguration()
	if metricsCfg.Namespace != "" {
		metrics.AddMetricsWithNamespace(metricsCfg.Namespace)
	}

	prepDbExitCode := prepareDB()
	if prepDbExitCode != ExitStatusOK {
		log.Info().Msgf(databasePreparationMessage, prepDbExitCode)
		return prepDbExitCode
	}

	ctx, cancel := context.WithCancel(context.Background())

	errorGroup := new(errgroup.Group)

	brokerConf := conf.GetBrokerConfiguration()
	// if broker is disabled, simply don't start it
	if brokerConf.Enabled {
		errorGroup.Go(func() error {
			defer cancel()

			err := startConsumer(brokerConf)
			if err != nil {
				log.Error().Err(err).Msg("Consumer start failure")
				return err
			}

			return nil
		})
	} else {
		log.Info().Msg("Broker is disabled, not starting it")
	}

	errorGroup.Go(func() error {
		defer cancel()

		err := startServer()
		if err != nil {
			log.Error().Err(err).Msg("Server start failure")
			return err
		}

		return nil
	})

	// it's gonna finish when either of goroutines finishes or fails
	<-ctx.Done()

	if errCode := stopService(); errCode != ExitStatusOK {
		return errCode
	}

	if err := errorGroup.Wait(); err != nil {
		// no need to log the error here since it's an error of the first failed goroutine
		return ExitStatusError
	}

	return ExitStatusOK
}

// stopService function stops the service and return error code indicating
// service status.
func stopService() int {
	errCode := ExitStatusOK

	err := stopServer()
	if err != nil {
		log.Error().Err(err).Msg("Server stop failure")
		errCode += ExitStatusServerError
	}

	brokerConf := conf.GetBrokerConfiguration()
	if brokerConf.Enabled {
		err = stopConsumer()
		if err != nil {
			log.Error().Err(err).Msg("Consumer stop failure")
			errCode += ExitStatusConsumerError
		}
	}

	return errCode
}

// initInfoLog is helper function to print value of one string parameter to
// logs.
func initInfoLog(msg string) {
	log.Info().Str(typeStr, "init").Msg(msg)
}

// printVersionInfo function prints basic information about service version
// into log file.
func printVersionInfo() {
	initInfoLog("Version: " + BuildVersion)
	initInfoLog("Build time: " + BuildTime)
	initInfoLog("Branch: " + BuildBranch)
	initInfoLog("Commit: " + BuildCommit)
	initInfoLog("Utils version:" + UtilsVersion)
}

const helpMessageTemplate = `
Aggregator service for insights results

Usage:

    %+v [command]

The commands are:

    <EMPTY>             starts aggregator
    start-service       starts aggregator
    help                prints help
    print-help          prints help
    print-config        prints current configuration set by files & env variables
    print-env           prints env variables
    print-version-info  prints version info
    migration           prints information about migrations (current, latest)
    migration <version> migrates database to the specified version

`

// printHelp function prints help to the standard output.
func printHelp() int {
	fmt.Printf(helpMessageTemplate, os.Args[0])
	return ExitStatusOK
}

// printConfig function prints the actual service configuration to the standard
// output.
func printConfig() int {
	configBytes, err := json.MarshalIndent(conf.Config, "", "    ")

	if err != nil {
		log.Error().Err(err).Msg("printConfig: marshall config failure")
		return 1
	}

	fmt.Println(string(configBytes))

	return ExitStatusOK
}

// printEnv function prints all environment variables to the standard output.
func printEnv() int {
	for _, keyVal := range os.Environ() {
		fmt.Println(keyVal)
	}

	return ExitStatusOK
}

// getDBForMigrations function opens a DB connection and prepares the DB for
// migrations. Non-OK exit code is returned as the last return value in case
// of an error. Otherwise, database and connection pointers are returned.
func getDBForMigrations() (storage.Storage, *sql.DB, int) {
	// use OCP recommendations storage only, unless migrations will be available for other storage(s) too
	var db storage.Storage

	ocpStorage, dvoStorage, err := createStorage()
	if err != nil {
		log.Error().Err(err).Msg("Unable to prepare DB for migrations")
		return nil, nil, ExitStatusPrepareDbError
	}

	backend := conf.GetStorageBackendConfiguration().Use
	switch backend {
	case types.OCPRecommendationsStorage:
		db = ocpStorage
	case types.DVORecommendationsStorage:
		db = dvoStorage
	default:
		log.Error().Msgf("storage backend %v does not support database migrations", db.GetDBDriverType())
		return nil, nil, ExitStatusMigrationError
	}

	dbConn := db.GetConnection()

	if err := migration.InitInfoTable(dbConn); err != nil {
		closeStorage(db)
		log.Error().Err(err).Msg("Unable to initialize migration info table")
		return nil, nil, ExitStatusPrepareDbError
	}

	return db, dbConn, ExitStatusOK
}

// printMigrationInfo function prints information about current DB migration
// version without making any modifications.
func printMigrationInfo(storage storage.Storage, dbConn *sql.DB) int {
	currMigVer, err := migration.GetDBVersion(dbConn)
	if err != nil {
		log.Error().Err(err).Msg("Unable to get current DB version")
		return ExitStatusMigrationError
	}

	log.Info().Msgf("Current DB version: %d", currMigVer)
	log.Info().Msgf("Maximum available version: %d", storage.GetMaxVersion())
	return ExitStatusOK
}

// setMigrationVersion function attempts to migrate the DB to the target
// version.
func setMigrationVersion(db storage.Storage, dbConn *sql.DB, versStr string) int {
	var targetVersion migration.Version

	if versStrLower := strings.ToLower(versStr); versStrLower == "latest" || versStrLower == "max" {
		targetVersion = db.GetMaxVersion()
	} else {
		vers, err := strconv.Atoi(versStr)
		if err != nil {
			log.Error().Err(err).Msg("Unable to parse target migration version")
			return ExitStatusMigrationError
		}

		targetVersion = migration.Version(vers)
	}

	if err := migration.SetDBVersion(dbConn, db.GetDBDriverType(), targetVersion, db.GetMigrations()); err != nil {
		log.Error().Err(err).Msg("Unable to perform migration")
		return ExitStatusMigrationError
	}

	log.Info().Msgf("Database version is now %d", targetVersion)
	return ExitStatusOK
}

// performMigrations function handles migrations subcommand. This can be used
// to either print the current DB migration version or to migrate to a
// different version.
func performMigrations() int {
	migrationArgs := os.Args[2:]

	db, dbConn, exitCode := getDBForMigrations()
	if exitCode != ExitStatusOK {
		return exitCode
	}
	defer closeStorage(db)

	switch len(migrationArgs) {
	case 0:
		return printMigrationInfo(db, dbConn)

	case 1:
		return setMigrationVersion(db, dbConn, migrationArgs[0])

	default:
		log.Error().Msg("Unexpected number of arguments to migrations command (expected 0-1)")
		return ExitStatusMigrationError
	}
}

func stopServiceOnProcessStopSignal() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		fmt.Println("SIGINT or SIGTERM was sent, stopping the service...")

		errCode := stopService()
		if errCode != 0 {
			log.Error().Msgf("unable to stop the service, code is %v", errCode)
			os.Exit(errCode)
		}
	}()
}

// main function represents entry point to the service.
func main() {
	err := conf.LoadConfiguration(defaultConfigFilename)
	if err != nil {
		panic(err)
	}

	err = logger.InitZerolog(
		conf.GetLoggingConfiguration(),
		conf.GetCloudWatchConfiguration(),
		conf.GetSentryLoggingConfiguration(),
		conf.GetKafkaZerologConfiguration(),
	)
	if err != nil {
		log.Error().Err(err).Msg("Unable to init ZeroLog")
	}

	command := "start-service"

	if len(os.Args) >= 2 {
		command = strings.ToLower(strings.TrimSpace(os.Args[1]))
	}

	errCode := handleCommand(command)
	if errCode != 0 {
		log.Error().Msgf("Service exited with non-zero code %v", errCode)
		os.Exit(errCode)
	}
}

// handleCommand function recognizes command provided via CLI and call the
// relevant code. Unknown commands are handled properly.
func handleCommand(command string) int {
	switch command {
	case "start-service":
		printVersionInfo()

		stopServiceOnProcessStopSignal()

		return startService()
	case "help", "print-help":
		return printHelp()
	case "print-config":
		return printConfig()
	case "print-env":
		return printEnv()
	case "print-version-info":
		printVersionInfo()
	case "migrations", "migration", "migrate":
		return performMigrations()
	default:
		fmt.Printf("\nCommand '%v' not found\n", command)
		return printHelp()
	}

	return ExitStatusOK
}
