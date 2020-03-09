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
	"log"
	"os"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

const (
	// ExitStatusOK means that the tool finished with success
	ExitStatusOK = iota
	// ExitStatusConsumerError is returned in case of any consumer-related error
	ExitStatusConsumerError
	// ExitStatusServerError is returned in case of any REST API server-related error
	ExitStatusServerError
	defaultConfigFilename = "config"
)

var (
	serverInstance   *server.HTTPServer
	consumerInstance consumer.Consumer
)

func startStorageConnection() (*storage.DBStorage, error) {
	storageCfg := getStorageConfiguration()

	dbStorage, err := storage.New(storageCfg)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return dbStorage, nil
}

// closeStorage closes specified DBStorage with proper error checking
// whether the close operation was successful or not.
func closeStorage(storage *storage.DBStorage) {
	err := storage.Close()
	if err != nil {
		log.Println("Error during closing storage connection", err)
	}
}

// closeConsumer closes specified consumer instance with proper error checking
// whether the close operation was successful or not.
func closeConsumer(consumerInstance consumer.Consumer) {
	err := consumerInstance.Close()
	if err != nil {
		log.Println("Error during closing consumer", err)
	}
}

// startConsumer starts consumer and returns exit code, 0 is no error
func startConsumer() int {
	dbStorage, err := startStorageConnection()
	if err != nil {
		return ExitStatusConsumerError
	}
	err = dbStorage.Init()
	if err != nil {
		log.Println(err)
		return ExitStatusConsumerError
	}
	defer closeStorage(dbStorage)

	brokerCfg := getBrokerConfiguration()

	// if broker is disabled, simply don't start it
	if !brokerCfg.Enabled {
		log.Println("Broker is disabled, not starting it")
		return ExitStatusOK
	}

	consumerInstance, err = consumer.New(brokerCfg, dbStorage)
	if err != nil {
		log.Println(err)
		return ExitStatusConsumerError
	}

	defer closeConsumer(consumerInstance)
	consumerInstance.Serve()

	return ExitStatusOK
}

// startServer starts the server and returns error code
func startServer() int {
	dbStorage, err := startStorageConnection()
	if err != nil {
		return ExitStatusServerError
	}
	defer closeStorage(dbStorage)

	serverCfg := getServerConfiguration()
	serverInstance = server.New(serverCfg, dbStorage)
	err = serverInstance.Start()
	if err != nil {
		log.Println(err)
		return ExitStatusServerError
	}

	return ExitStatusOK
}

// startService starts service and returns error code
func startService() int {
	var waitGroup sync.WaitGroup
	exitCode := 0

	waitGroup.Add(1)
	// consumer is run in its own thread
	go func() {
		consumerExitCode := startConsumer()
		if consumerExitCode != 0 {
			fmt.Printf("consumer exited with error code %v", consumerExitCode)
			exitCode += consumerExitCode
		}

		waitGroup.Done()
	}()

	// server can be started in current thread
	serverExitCode := startServer()
	if serverExitCode != 0 {
		fmt.Printf("consumer exited with error code %v", serverExitCode)
		exitCode += serverExitCode
	}

	waitGroup.Wait()

	return exitCode
}

func waitForServiceToStart() {
	for {
		isStarted := true
		if viper.Sub("broker").GetBool("enabled") && consumerInstance == nil {
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
			log.Println(err)
			errCode++
		}
	}

	if consumerInstance != nil {
		err := consumerInstance.Close()
		if err != nil {
			log.Println(err)
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
}
