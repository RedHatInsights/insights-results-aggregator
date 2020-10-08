package main

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/server"
)

var (
	serverInstance                                               *server.HTTPServer
	serverInstanceIsStarting, finishServerInstanceInitialization = context.WithCancel(context.Background())
)

// startServer starts the server or returns an error
func startServer() error {
	defer func() {
		finishServerInstanceInitialization()
	}()

	dbStorage, err := createStorage()
	if err != nil {
		return err
	}
	defer closeStorage(dbStorage)

	serverCfg := conf.GetServerConfiguration()

	serverInstance = server.New(serverCfg, dbStorage)

	err = serverInstance.Start(finishServerInstanceInitialization)
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) start error")
		return err
	}

	return nil
}

func stopServer() error {
	waitForServerToStartOrFail()

	if serverInstance == nil {
		return nil
	}

	err := serverInstance.Stop(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) server stop error")
		return err
	}

	return nil
}

func waitForServerToStartOrFail() {
	log.Info().Msg("waiting for server to start")
	_ = <-serverInstanceIsStarting.Done()
}
