// Copyright 2020, 2021 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
)

var (
	consumerInstance                                                 *consumer.KafkaConsumer
	consumerInstanceIsStarting, finishConsumerInstanceInitialization = context.WithCancel(context.Background())
)

// startConsumer function starts the consumer or returns an error in case of
// any error. When consumer is started properly, nil is returned instead.
func startConsumer(brokerConf broker.Configuration) error {
	defer finishConsumerInstanceInitialization()

	dbStorage, err := createStorage()
	if err != nil {
		return err
	}

	defer closeStorage(dbStorage)

	consumerInstance, err = consumer.New(brokerConf, dbStorage)
	if err != nil {
		log.Error().Err(err).Msg("Broker initialization error")
		return err
	}

	finishConsumerInstanceInitialization()
	consumerInstance.Serve()

	return nil
}

// stopConsumer function tries to stop the consumer. If consumer is not started
// or if it is not possible to stop it properly, error value is returned.
func stopConsumer() error {
	waitForConsumerToStartOrFail()

	if consumerInstance == nil {
		return nil
	}

	err := consumerInstance.Close()
	if err != nil {
		log.Error().Err(err).Msg("Consumer stop error")
		return err
	}

	return nil
}

// waitForConsumerToStartOrFail function perform synchronization with the
// consumer goroutine via shared channel.
func waitForConsumerToStartOrFail() {
	log.Info().Msg("waiting for consumer to start")
	<-consumerInstanceIsStarting.Done()
}
