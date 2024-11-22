package main

import (
	"context"
	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

var (
	heartbeatConsumerInstance                                                          *consumer.KfkConsumer
	heartbeatConsumerInstanceIsStarting, heartbeatConsumerInstanceFinishInitialization = context.WithCancel(context.Background())
)

func stopHeartbeatConsumer() int {
	waitForHeartbeatConsumerToStartOrFail()

	if heartbeatConsumerInstance == nil {
		return ExitStatusOK
	}

	err := heartbeatConsumerInstance.Close()
	if err != nil {
		log.Error().Err(err).Msg("Consumer stop error")
		return ExitStatusConsumerError
	}

	return ExitStatusOK
}

func waitForHeartbeatConsumerToStartOrFail() {
	log.Info().Msg("waiting for heartbeat consumer to start")
	<-heartbeatConsumerInstanceIsStarting.Done()
}

func startHeartbeatConsumer() int {
	defer heartbeatConsumerInstanceFinishInitialization()

	// general setup
	setupErrCode := setupService()
	if setupErrCode != 0 {
		return setupErrCode
	}

	// right now just the OCP recommendation storage is handled by consumer
	_, dvoRecommendationStorage, err := createStorage()
	if err != nil {
		return ExitStatusError
	}

	defer closeStorage(dvoRecommendationStorage)

	heartbeatMessageProcessor := consumer.HearbeatMessageProcessor{Storage: dvoRecommendationStorage}
	brokerConf := conf.GetBrokerConfiguration()

	// when heartbeats consumer will be made, it will need to use DVO storage
	// (see line that calls createStorage())
	if conf.GetStorageBackendConfiguration().Use == types.DVORecommendationsStorage {
		heartbeatConsumerInstance, err = consumer.NewKfkConsumer(brokerConf, &heartbeatMessageProcessor)
		if err != nil {
			log.Error().Err(err).Msg("Hearbeats consumer initialization error")
			return ExitStatusError
		}
	} else {
		log.Error().Msg("Expecting to use DVO backend for heatbeats. Not found. Exitting")
		return ExitStatusError
	}

	heartbeatConsumerInstanceFinishInitialization()

	ctx := context.Background()
	heartbeatConsumerInstance.Serve(ctx)

	return ExitStatusOK
}
