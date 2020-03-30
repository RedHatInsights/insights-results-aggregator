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
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/content"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
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
	defaultConfigFilename = "config"

	databasePreparationMessage = "database preparation existed with error code %v"
	consumerExitedErrorMessage = "consumer exited with error code %v"
)

var (
	serverInstance   *server.HTTPServer
	consumerInstance consumer.Consumer
)

func createStorage() (*storage.DBStorage, error) {
	storageCfg := getStorageConfiguration()

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

// prepareDB migrates the DB to the latest version
// and loads all available rule content into it.
func prepareDB() int {
	dbStorage, err := createStorage()
	if err != nil {
		return ExitStatusPrepareDbError
	}
	defer closeStorage(dbStorage)

	// Initialize the database by running necessary
	// migrations to get to the highest available version.
	err = dbStorage.Init()
	if err != nil {
		log.Error().Err(err).Msg("DB initialization error")
		return ExitStatusPrepareDbError
	}

	ruleContentDirPath := getContentPathConfiguration()
	contentDir, err := content.ParseRuleContentDir(ruleContentDirPath)
	if osPathError, ok := err.(*os.PathError); ok {
		log.Error().Err(osPathError).Msg("No rules directory")
		return ExitStatusPrepareDbError
	}

	if err := dbStorage.LoadRuleContent(contentDir); err != nil {
		log.Error().Err(err).Msg("Rules content loading error")
		return ExitStatusPrepareDbError
	}

	return ExitStatusOK
}

// startConsumer starts consumer and returns exit code, 0 is no error
func startConsumer() int {
	dbStorage, err := createStorage()
	if err != nil {
		return ExitStatusConsumerError
	}
	defer closeStorage(dbStorage)

	brokerCfg := getBrokerConfiguration()

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

	serverCfg := getServerConfiguration()
	serverInstance = server.New(serverCfg, dbStorage)
	err = serverInstance.Start()
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) start error")
		return ExitStatusServerError
	}

	return ExitStatusOK
}

// startService starts service and returns error code
func startService() int {
	var waitGroup sync.WaitGroup
	exitCode := 0

	prepDbExitCode := prepareDB()
	if prepDbExitCode != 0 {
		log.Info().Msg(fmt.Sprintf(databasePreparationMessage, prepDbExitCode))
		exitCode += prepDbExitCode
		return exitCode
	}

	waitGroup.Add(1)
	// consumer is run in its own thread
	go func() {
		consumerExitCode := startConsumer()
		if consumerExitCode != 0 {
			log.Info().Msg(fmt.Sprintf(consumerExitedErrorMessage, prepDbExitCode))
			exitCode += consumerExitCode
		}

		waitGroup.Done()
	}()

	// server can be started in current thread
	serverExitCode := startServer()
	if serverExitCode != 0 {
		log.Info().Msg(fmt.Sprintf(consumerExitedErrorMessage, prepDbExitCode))
		exitCode += serverExitCode
	}

	waitGroup.Wait()

	return exitCode
}

func waitForServiceToStart() {
	for {
		isStarted := true
		if getBrokerConfiguration().Enabled && consumerInstance == nil {
			isStarted = false
		}
		if serverInstance == nil {
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
	errCode := 0

	if serverInstance != nil {
		err := serverInstance.Stop(context.TODO())
		if err != nil {
			log.Error().Err(err).Msg("HTTP(s) server stop error")
			errCode++
		}
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

func main() {
	err := loadConfiguration(defaultConfigFilename)
	if err != nil {
		panic(err)
	}

	errCode := startService()
	if errCode != 0 {
		os.Exit(errCode)
	}

	errCode = stopService()
	if errCode != 0 {
		os.Exit(errCode)
	}
}
