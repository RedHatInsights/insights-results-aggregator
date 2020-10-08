package main

import (
	"context"
	"reflect"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
)

var (
	consumerInstance                                                 consumer.Consumer
	consumerInstanceIsStarting, finishConsumerInstanceInitialization = context.WithCancel(context.Background())
)

// startConsumer starts the consumer or returns an error
func startConsumer() error {
	defer func() {
		finishConsumerInstanceInitialization()
	}()

	dbStorage, err := createStorage()
	if err != nil {
		return err
	}

	defer closeStorage(dbStorage)

	brokerCfg := conf.GetBrokerConfiguration()
	// if broker is disabled, simply don't start it
	if !brokerCfg.Enabled {
		log.Info().Msg("Broker is disabled, not starting it")
		return nil
	}

	consumerInstance, err = consumer.New(brokerCfg, dbStorage)
	if err != nil {
		log.Error().Err(err).Msg("Broker initialization error")
		return nil
	}

	finishConsumerInstanceInitialization()
	consumerInstance.Serve()

	return nil
}

func stopConsumer() error {
	waitForConsumerToStartOrFail()

	if reflect.ValueOf(consumerInstance).IsNil() {
		return nil
	}

	err := consumerInstance.Close()
	if err != nil {
		log.Error().Err(err).Msg("Consumer stop error")
		return err
	}

	return nil
}

func waitForConsumerToStartOrFail() {
	log.Info().Msg("waiting for consumer to start")
	_ = <-consumerInstanceIsStarting.Done()
}
