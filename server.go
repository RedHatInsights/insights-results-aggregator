// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var (
	serverInstance                                               *server.HTTPServer
	serverInstanceIsStarting, finishServerInstanceInitialization = context.WithCancel(context.Background())
)

// startServer starts the server or returns an error
func startServer() error {
	defer finishServerInstanceInitialization()

	ocpRecommendationsStorage, dvoRecommendationsStorage, err := createStorage()
	if err != nil {
		return err
	}
	defer closeStorage(ocpRecommendationsStorage)
	defer closeStorage(dvoRecommendationsStorage)

	serverCfg := conf.GetServerConfiguration()

	serverInstance = server.New(serverCfg, ocpRecommendationsStorage, dvoRecommendationsStorage)

	// fill-in additional info used by /info endpoint handler
	fillInInfoParams(serverInstance.InfoParams)

	// try to retrieve the actual DB migration version
	// and add it into the `params` map
	log.Info().Msg("Setting OCP DB version for /info endpoint")
	setDBVersion(
		serverInstance,
		conf.GetOCPRecommendationsStorageConfiguration(),
		ocpRecommendationsStorage,
		"OCP_DB_version",
	)
	log.Info().Msg("Setting DVO DB version for /info endpoint")
	setDBVersion(
		serverInstance,
		conf.GetDVORecommendationsStorageConfiguration(),
		dvoRecommendationsStorage,
		"DVO_DB_version",
	)

	err = serverInstance.Start(finishServerInstanceInitialization)
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) start error")
		return err
	}

	return nil
}

func setDBVersion(s *server.HTTPServer, storageConf storage.Configuration, storage storage.Storage, key string) {
	if storageConf.Type != types.SQLStorage {
		// migration and DB versioning is now supported for SQL databases only
		s.InfoParams[key] = "not supported"
		return
	}
	currentVersion, err := migration.GetDBVersion(storage.GetConnection(), storage.GetDBSchema())
	if err != nil {
		const msg = "Unable to retrieve DB migration version"
		log.Error().Err(err).Msg(msg)
		serverInstance.InfoParams[key] = msg
	} else {
		serverInstance.InfoParams[key] = strconv.FormatUint(uint64(currentVersion), 10)
	}
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
	<-serverInstanceIsStarting.Done()
}
