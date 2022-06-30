// Copyright 2020, 2021, 2022 Red Hat, Inc
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

package consumer_test

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func benchmarkProcessingMessage(b *testing.B, s storage.Storage, messageProducer func() string) {
	kafkaConsumer := &consumer.KafkaConsumer{
		Storage: s,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mustConsumerProcessMessage(b, kafkaConsumer, messageProducer())
	}
}

func getNoopStorage(testing.TB, bool) (storage.Storage, func()) {
	return &storage.NoopStorage{}, func() {}
}

func BenchmarkKafkaConsumer_ProcessMessage_SimpleMessages(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	var testCases = []struct {
		Name            string
		StorageProducer func(testing.TB, bool) (storage.Storage, func())
		RandomMessages  bool
	}{
		{"NoopStorage", getNoopStorage, false},
		{"NoopStorage", getNoopStorage, true},
		{"SQLiteInMemory", ira_helpers.MustGetSQLiteMemoryStorage, false},
		{"SQLiteInMemory", ira_helpers.MustGetSQLiteMemoryStorage, true},
		{"Postgres", ira_helpers.MustGetPostgresStorage, false},
		{"Postgres", ira_helpers.MustGetPostgresStorage, true},
		{"SQLiteFile", ira_helpers.MustGetSQLiteFileStorage, false},
		{"SQLiteFile", ira_helpers.MustGetSQLiteFileStorage, true},
	}

	for _, testCase := range testCases {
		if testCase.RandomMessages {
			testCase.Name += "/RandomMessages"
		}

		b.Run(testCase.Name, func(b *testing.B) {
			benchStorage, cleaner := testCase.StorageProducer(b, true)
			if cleaner != nil {
				defer cleaner()
			}
			defer ira_helpers.MustCloseStorage(b, benchStorage)

			if testCase.RandomMessages {
				benchmarkProcessingMessage(b, benchStorage, testdata.GetRandomConsumerMessage)
			} else {
				benchmarkProcessingMessage(b, benchStorage, func() string {
					return testdata.ConsumerMessage
				})
			}
		})
	}
}

func getMessagesFromDir(b *testing.B, dataDir string) []string {
	files, err := os.ReadDir(dataDir)
	helpers.FailOnError(b, err)

	var messages []string

	for _, file := range files {
		if file.Mode().IsRegular() {
			if strings.HasSuffix(file.Name(), ".json") && !strings.Contains(file.Name(), "broken") {
				filePath := path.Join(dataDir, file.Name())

				fileBytes, err := os.ReadFile(filePath)
				helpers.FailOnError(b, err)

				zerolog.SetGlobalLevel(zerolog.Disabled)
				parsedMessage, err := consumer.ParseMessage(fileBytes)
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
				if err != nil {
					log.Warn().Msgf("skipping file %+v because it has bad structure", file.Name())
					continue
				}
				err = consumer.CheckReportStructure(*parsedMessage.Report)
				if err != nil {
					log.Warn().Msgf("skipping file %+v because its report has bad structure", file.Name())
					continue
				}

				fileContent := string(fileBytes)

				messages = append(messages, fileContent)
			}
		}
	}

	return messages
}

func BenchmarkKafkaConsumer_ProcessMessage_RealMessages(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	messages := getMessagesFromDir(b, "../utils/produce_insights_results/")

	var testCases = []struct {
		Name            string
		StorageProducer func(testing.TB, bool) (storage.Storage, func())
	}{
		{"NoopStorage", getNoopStorage},
		{"SQLiteInMemory", ira_helpers.MustGetSQLiteMemoryStorage},
		{"Postgres", ira_helpers.MustGetPostgresStorage},
		{"SQLiteFile", ira_helpers.MustGetSQLiteFileStorage},
	}

	for _, testCase := range testCases {
		testCase.Name += "/" + testCase.Name

		b.Run(testCase.Name, func(b *testing.B) {
			benchStorage, cleaner := testCase.StorageProducer(b, true)
			if cleaner != nil {
				defer cleaner()
			}
			defer ira_helpers.MustCloseStorage(b, benchStorage)

			kafkaConsumer := &consumer.KafkaConsumer{
				Storage: benchStorage,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, message := range messages {
					mustConsumerProcessMessage(b, kafkaConsumer, message)
				}
			}
		})
	}
}
