// Copyright 2020 Red Hat, Inc
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

package logger_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/logger"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestSaramaZerologger(t *testing.T) {
	const expectedErrStrInfoLevel = "some random error message"
	const expectedErrStrErrorLevel = "kafka: error test error"

	buf := new(bytes.Buffer)

	err := logger.InitZerolog(
		logger.LoggingConfiguration{
			Debug:                      false,
			LogLevel:                   "debug",
			LoggingToCloudWatchEnabled: false,
		},
		logger.CloudWatchConfiguration{},
		zerolog.New(buf),
	)
	helpers.FailOnError(t, err)

	t.Run("InfoLevel", func(t *testing.T) {
		buf.Reset()

		sarama.Logger.Printf(expectedErrStrInfoLevel)

		assert.Contains(t, buf.String(), `\"level\":\"info\"`)
		assert.Contains(t, buf.String(), expectedErrStrInfoLevel)
	})

	t.Run("ErrorLevel", func(t *testing.T) {
		buf.Reset()

		sarama.Logger.Print(expectedErrStrErrorLevel)

		assert.Contains(t, buf.String(), `\"level\":\"error\"`)
		assert.Contains(t, buf.String(), expectedErrStrErrorLevel)
	})
}

func TestLoggerSetLogLevel(t *testing.T) {
	logLevels := []string{"debug", "info", "warning", "error"}
	for logLevelIndex, logLevel := range logLevels {
		t.Run(logLevel, func(t *testing.T) {
			buf := new(bytes.Buffer)

			err := logger.InitZerolog(
				logger.LoggingConfiguration{
					Debug:                      false,
					LogLevel:                   logLevel,
					LoggingToCloudWatchEnabled: false,
				},
				logger.CloudWatchConfiguration{},
				zerolog.New(buf),
			)
			helpers.FailOnError(t, err)

			log.Debug().Msg("debug level")
			log.Info().Msg("info level")
			log.Warn().Msg("warning level")
			log.Error().Msg("error level")

			for i := 0; i < len(logLevels); i++ {
				if i < logLevelIndex {
					assert.NotContains(t, buf.String(), logLevels[i]+" level")
				} else {
					assert.Contains(t, buf.String(), logLevels[i]+" level")
				}
			}
		})
	}
}

func TestUnJSONWriter_Write(t *testing.T) {
	for _, testCase := range []struct {
		Name        string
		StrToWrite  string
		ExpectedStr string
	}{
		{"NotJSON", "some expected string", "some expected string"},
		{"JSON", `{"level": "error", "is_something": true}`, "LEVEL=error; IS_SOMETHING=true;"},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			unJSONWriter := logger.UnJSONWriter{Writer: buf}

			writtenBytes, err := unJSONWriter.Write([]byte(testCase.StrToWrite))
			helpers.FailOnError(t, err)

			assert.Equal(t, writtenBytes, len(testCase.StrToWrite))
			assert.Equal(t, testCase.ExpectedStr, strings.TrimSpace(buf.String()))
		})
	}
}

func TestInitZerolog_LogToCloudWatch(t *testing.T) {
	// TODO: mock logging to cloud watch and do actual testing
	err := logger.InitZerolog(
		logger.LoggingConfiguration{
			Debug:                      false,
			LogLevel:                   "debug",
			LoggingToCloudWatchEnabled: true,
		},
		logger.CloudWatchConfiguration{},
	)
	helpers.FailOnError(t, err)
}
