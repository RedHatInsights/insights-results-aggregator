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
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/logger"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	// ExitStatusOK means that the tool finished with success
	ExitStatusOK = iota
	// ExitStatusPrepareDbError is returned when the DB preparation (including rule content loading) fails
	ExitStatusPrepareDbError
	// ExitStatusConsumerError is returned in case of any consumer-related error
	ExitStatusConsumerError
	// ExitStatusServerError is returned in case of any REST API server-related error
	ExitStatusServerError
	// ExitStatusMigrationError is returned in case of an error while attempting to perform DB migrations
	ExitStatusMigrationError
	defaultConfigFilename = "config"
	typeStr               = "type"

	databasePreparationMessage = "database preparation exited with error code %v"
	consumerExitedErrorMessage = "consumer exited with error code %v"
)

var (
	serverInstance      *server.HTTPServer
	consumerInstance    consumer.Consumer
	serverInstanceReady = sync.NewCond(&sync.Mutex{})

	// BuildVersion contains the major.minor version of the CLI client
	BuildVersion = "*not set*"

	// BuildTime contains timestamp when the CLI client has been built
	BuildTime = "*not set*"

	// BuildBranch contains Git branch used to build this application
	BuildBranch = "*not set*"

	// BuildCommit contains Git commit used to build this application
	BuildCommit = "*not set*"

	// autoMigrate determines if the prepareDB function upgrades
	// the database to the latest migration version. This is necessary
	// for certain tests that work with a temporary, empty SQLite DB.
	autoMigrate = false
)

func createStorage() (*storage.DBStorage, error) {
	storageCfg := conf.GetStorageConfiguration()

	dbStorage, err := storage.New(storageCfg)
	if err != nil {
		log.Error().Err(err).Msg("storage.New")
		return nil, err
	}

	return dbStorage, nil
}

// closeStorage closes specified DBStorage with proper error checking
// whether the close operation was successful or not.
func closeStorage(storage *storage.DBStorage) {
	err := storage.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error during closing storage connection")
	}
}

func prepareDBMigrations(dbStorage *storage.DBStorage) int {
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

		maxVersion := migration.GetMaxVersion()
		if currentVersion != maxVersion {
			log.Error().Msgf("old DB migration version (current: %d, latest: %d)", currentVersion, maxVersion)
			return ExitStatusPrepareDbError
		}
	}

	return ExitStatusOK
}

// prepareDB opens a DB connection and loads all available rule content into it.
func prepareDB() int {
	dbStorage, err := createStorage()
	if err != nil {
		return ExitStatusPrepareDbError
	}
	defer closeStorage(dbStorage)

	// Ensure that the DB is at the latest migration version.
	if exitCode := prepareDBMigrations(dbStorage); exitCode != ExitStatusOK {
		return exitCode
	}

	// Initialize the database.
	err = dbStorage.Init()
	if err != nil {
		log.Error().Err(err).Msg("DB initialization error")
		return ExitStatusPrepareDbError
	}

	return ExitStatusOK
}

// startConsumer starts consumer and returns exit code, ExitStatusOK is no error
func startConsumer() int {
	dbStorage, err := createStorage()
	if err != nil {
		return ExitStatusConsumerError
	}
	defer closeStorage(dbStorage)

	brokerCfg := conf.GetBrokerConfiguration()

	// if broker is disabled, simply don't start it
	if !brokerCfg.Enabled {
		log.Info().Msg("Broker is disabled, not starting it")
		return ExitStatusOK
	}

	consumerInstance, err = consumer.New(brokerCfg, dbStorage)
	if err != nil {
		log.Error().Err(err).Msg("Broker initialization error")
		return ExitStatusConsumerError
	}

	consumerInstance.Serve()

	return ExitStatusOK
}

// startServer starts the server and returns error code
func startServer() int {
	dbStorage, err := createStorage()
	if err != nil {
		return ExitStatusServerError
	}
	defer closeStorage(dbStorage)

	serverCfg := conf.GetServerConfiguration()

	s := server.New(serverCfg, dbStorage)

	serverInstanceReady.L.Lock()
	serverInstance = s
	serverInstanceReady.L.Unlock()

	err = serverInstance.Start(serverInstanceReady)
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) start error")
		return ExitStatusServerError
	}

	return ExitStatusOK
}

// startService starts service and returns error code
func startService() int {
	var waitGroup sync.WaitGroup
	exitCode := int32(ExitStatusOK)

	metricsCfg := conf.GetMetricsConfiguration()
	if metricsCfg.Namespace != "" {
		metrics.AddMetricsWithNamespace(metricsCfg.Namespace)
	}

	prepDbExitCode := prepareDB()
	if prepDbExitCode != ExitStatusOK {
		log.Info().Msgf(databasePreparationMessage, prepDbExitCode)
		exitCode += int32(prepDbExitCode)

		return int(exitCode)
	}

	waitGroup.Add(1)
	// consumer is run in its own thread
	go func() {
		defer waitGroup.Done()

		consumerExitCode := startConsumer()
		if consumerExitCode != ExitStatusOK {
			log.Info().Msgf(consumerExitedErrorMessage, prepDbExitCode)
			atomic.AddInt32(&exitCode, int32(consumerExitCode))
		}
	}()

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()

		// server can be started in current thread
		serverExitCode := startServer()
		if serverExitCode != ExitStatusOK {
			log.Info().Msgf(consumerExitedErrorMessage, prepDbExitCode)
			atomic.AddInt32(&exitCode, int32(serverExitCode))
		}
	}()

	waitGroup.Wait()

	return int(exitCode)
}

func waitForServiceToStart() {
	serverInstanceReady.L.Lock()
	if serverInstance == nil {
		serverInstanceReady.Wait()
	}
	serverInstanceReady.L.Unlock()

	for {
		isStarted := true
		if conf.GetBrokerConfiguration().Enabled && consumerInstance == nil {
			isStarted = false
		}

		if isStarted {
			// everything was initialized
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func stopService() int {
	errCode := ExitStatusOK

	serverInstanceReady.L.Lock()
	if serverInstance == nil {
		serverInstanceReady.Wait()
	}
	serverInstanceReady.L.Unlock()

	err := serverInstance.Stop(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) server stop error")
		errCode++
	}

	if consumerInstance != nil {
		err := consumerInstance.Close()
		if err != nil {
			log.Error().Err(err).Msg("Consumer stop error")
			errCode++
		}
	}

	return errCode
}

func initInfoLog(msg string) {
	log.Info().Str(typeStr, "init").Msg(msg)
}

func printVersionInfo() {
	initInfoLog("Version: " + BuildVersion)
	initInfoLog("Build time: " + BuildTime)
	initInfoLog("Branch: " + BuildBranch)
	initInfoLog("Commit: " + BuildCommit)
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

func printHelp() int {
	fmt.Printf(helpMessageTemplate, os.Args[0])
	return ExitStatusOK
}

func printConfig() int {
	configBytes, err := json.MarshalIndent(conf.Config, "", "    ")

	if err != nil {
		log.Error().Err(err)
		return 1
	}

	fmt.Println(string(configBytes))

	return ExitStatusOK
}

func printEnv() int {
	for _, keyVal := range os.Environ() {
		fmt.Println(keyVal)
	}

	return ExitStatusOK
}

// getDBForMigrations opens a DB connection and prepares the DB for migrations.
// Non-OK exit code is returned as the last return value in case of an error.
// Otherwise, database and connection pointers are returned.
func getDBForMigrations() (*storage.DBStorage, *sql.DB, int) {
	db, err := createStorage()
	if err != nil {
		log.Error().Err(err).Msg("Unable to prepare DB for migrations")
		return nil, nil, ExitStatusPrepareDbError
	}

	dbConn := db.GetConnection()

	if err := migration.InitInfoTable(dbConn); err != nil {
		closeStorage(db)
		log.Error().Err(err).Msg("Unable to initialize migration info table")
		return nil, nil, ExitStatusPrepareDbError
	}

	return db, dbConn, ExitStatusOK
}

// printMigrationInfo prints information about current DB
// migration version without making any modifications.
func printMigrationInfo(dbConn *sql.DB) int {
	currMigVer, err := migration.GetDBVersion(dbConn)
	if err != nil {
		log.Error().Err(err).Msg("Unable to get current DB version")
		return ExitStatusMigrationError
	}

	log.Info().Msgf("Current DB version: %d", currMigVer)
	log.Info().Msgf("Maximum available version: %d", migration.GetMaxVersion())
	return ExitStatusOK
}

// setMigrationVersion attempts to migrate the DB to the target version.
func setMigrationVersion(dbConn *sql.DB, dbDriver types.DBDriver, versStr string) int {
	var targetVersion migration.Version
	if versStrLower := strings.ToLower(versStr); versStrLower == "latest" || versStrLower == "max" {
		targetVersion = migration.GetMaxVersion()
	} else {
		vers, err := strconv.Atoi(versStr)
		if err != nil {
			log.Error().Err(err).Msg("Unable to parse target migration version")
			return ExitStatusMigrationError
		}

		targetVersion = migration.Version(vers)
	}

	if err := migration.SetDBVersion(dbConn, dbDriver, targetVersion); err != nil {
		log.Error().Err(err).Msg("Unable to perform migration")
		return ExitStatusMigrationError
	}

	log.Info().Msgf("Database version is now %d", targetVersion)
	return ExitStatusOK
}

// performMigrations handles migrations subcommand. This can be used to either
// print the current DB migration version or to migrate to a different version.
func performMigrations() int {
	migrationArgs := os.Args[2:]

	db, dbConn, exitCode := getDBForMigrations()
	if exitCode != ExitStatusOK {
		return exitCode
	}
	defer closeStorage(db)

	switch len(migrationArgs) {
	case 0:
		return printMigrationInfo(dbConn)

	case 1:
		return setMigrationVersion(dbConn, db.GetDBDriverType(), migrationArgs[0])

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

		serverInstanceReady.L.Lock()
		if serverInstance == nil {
			serverInstanceReady.Wait()
		}
		serverInstanceReady.L.Unlock()

		errCode := stopService()
		if errCode != 0 {
			log.Error().Msgf("unable to stop the service, code is %v", errCode)
			os.Exit(errCode)
		}
	}()
}

func main() {
	err := conf.LoadConfiguration(defaultConfigFilename)
	if err != nil {
		panic(err)
	}

	err = logger.InitZerolog(conf.GetLoggingConfiguration(), conf.GetCloudWatchConfiguration())
	if err != nil {
		panic(err)
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
