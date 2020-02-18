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
// The service contains consumer (usually Kafka consumer) that consume
// messages from given source, processs those messages, and stores them
// in configured data store. It also starts REST API servers with
// endpoints that expose several types of information: list of organizations,
// list of clusters for given organization, and cluster health.
package main

import (
	"context"
	"log"
	"os"
	"time"

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
)

var (
	serverInstance   *server.HTTPServer = nil
	consumerInstance consumer.Consumer  = nil
)

func startStorageConnection() (*storage.DBStorage, error) {
	storageCfg := loadStorageConfiguration()
	storage, err := storage.New(storageCfg)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return storage, nil
}

func startConsumer() {
	storage, err := startStorageConnection()
	if err != nil {
		os.Exit(ExitStatusConsumerError)
	}
	err = storage.Init()
	if err != nil {
		log.Println(err)
		os.Exit(ExitStatusConsumerError)
	}
	defer storage.Close()

	brokerCfg := loadBrokerConfiguration()

	// if broker is disabled, simply don't start it
	if !brokerCfg.Enabled {
		log.Println("Broker is disabled, not starting it")
		return
	}

	consumerInstance, err = consumer.New(brokerCfg, storage)
	if err != nil {
		log.Println(err)
		os.Exit(ExitStatusConsumerError)
	}
	defer consumerInstance.Close()
	err = consumerInstance.Start()
	if err != nil {
		log.Println(err)
		os.Exit(ExitStatusConsumerError)
	}
}

func startServer() {
	storage, err := startStorageConnection()
	if err != nil {
		os.Exit(ExitStatusServerError)
	}
	defer storage.Close()

	serverCfg := loadServerConfiguration()
	serverInstance = server.New(serverCfg, storage)
	err = serverInstance.Start()
	if err != nil {
		log.Println(err)
		os.Exit(ExitStatusServerError)
	}
}

func startService() {
	// consumer is run in its own thread
	go startConsumer()
	// server can be started in current thread
	startServer()
	os.Exit(ExitStatusOK)
}

func waitForServiceToStart() {
	for {
		if serverInstance != nil && consumerInstance != nil {
			// everything was initialized
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func stopService() {
	err := serverInstance.Stop(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	err = consumerInstance.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	loadConfiguration("config")

	startService()
}
