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
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/logger"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

const (
	testTimeout = 10 * time.Second
)

var (
	cloudWatchConf = logger.CloudWatchConfiguration{
		AWSAccessID:             "access ID",
		AWSSecretKey:            "secret",
		AWSSessionToken:         "sess token",
		AWSRegion:               "aws region",
		LogGroup:                "log group",
		StreamName:              "stream name",
		CreateStreamIfNotExists: true,
		Debug:                   false,
	}
	describeLogStreamsEvent = CloudWatchExpect{
		http.MethodPost,
		"Logs_20140328.DescribeLogStreams",
		`{
					"descending": false,
					"logGroupName": "` + cloudWatchConf.LogGroup + `",
					"logStreamNamePrefix": "` + cloudWatchConf.StreamName + `",
					"orderBy": "LogStreamName"
				}`,
		http.StatusOK,
		`{
					"logStreams": [
						{
							"arn": "arn:aws:logs:` +
			cloudWatchConf.AWSRegion + `:012345678910:log-group:` + cloudWatchConf.LogGroup +
			`:log-stream:` + cloudWatchConf.StreamName + `",
							"creationTime": 1,
							"firstEventTimestamp": 2,
							"lastEventTimestamp": 3,
							"lastIngestionTime": 4,
							"logStreamName": "` + cloudWatchConf.StreamName + `",
							"storedBytes": 100,
							"uploadSequenceToken": "1"
						}
					],
					"nextToken": "token1"
				}`,
	}
)

type CloudWatchExpect struct {
	ExpectedMethod   string
	ExpectedTarget   string
	ExpectedBody     string
	ResultStatusCode int
	ResultBody       string
}

func TestSaramaZerologger(t *testing.T) {
	const expectedStrInfoLevel = "some random message"
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

		sarama.Logger.Printf(expectedStrInfoLevel)

		assert.Contains(t, buf.String(), `\"level\":\"info\"`)
		assert.Contains(t, buf.String(), expectedStrInfoLevel)
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

func TestWorkaroundForRHIOPS729_Write(t *testing.T) {
	for _, testCase := range []struct {
		Name        string
		StrToWrite  string
		ExpectedStr string
		IsJSON      bool
	}{
		{"NotJSON", "some expected string", "some expected string", false},
		{
			"JSON",
			`{"level": "error", "is_something": true}`,
			`{"LEVEL":"error", "IS_SOMETHING": true}`,
			true,
		},
	} {
		t.Run(testCase.Name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			unJSONWriter := logger.WorkaroundForRHIOPS729{Writer: buf}

			writtenBytes, err := unJSONWriter.Write([]byte(testCase.StrToWrite))
			helpers.FailOnError(t, err)

			assert.Equal(t, writtenBytes, len(testCase.StrToWrite))
			if testCase.IsJSON {
				helpers.AssertStringsAreEqualJSON(t, testCase.ExpectedStr, buf.String())
			} else {
				assert.Equal(t, testCase.ExpectedStr, strings.TrimSpace(buf.String()))
			}
		})
	}
}

func TestInitZerolog_LogToCloudWatch(t *testing.T) {
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

func TestLoggingToCloudwatch(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		const baseURL = "http://localhost:9999"
		logger.AWSCloudWatchEndpoint = baseURL + "/cloudwatch"

		expects := []CloudWatchExpect{
			{
				http.MethodPost,
				"Logs_20140328.CreateLogStream",
				`{
					"logGroupName": "` + cloudWatchConf.LogGroup + `",
					"logStreamName": "` + cloudWatchConf.StreamName + `"
				}`,
				http.StatusBadRequest,
				`{
					"__type": "ResourceAlreadyExistsException",
					"message": "The specified log stream already exists"
				}`,
			},
			describeLogStreamsEvent,
			{
				http.MethodPost,
				"Logs_20140328.PutLogEvents",
				`{
					"logEvents": [
						{
							"message": "test message text goes right here",
							"timestamp": 1
						}
					],
					"logGroupName": "` + cloudWatchConf.LogGroup + `",
					"logStreamName":"` + cloudWatchConf.StreamName + `",
					"sequenceToken":"1"
				}`,
				http.StatusOK,
				`{"nextSequenceToken":"2"}`,
			},
			{
				http.MethodPost,
				"Logs_20140328.PutLogEvents",
				`{
					"logEvents": [
						{
							"message": "second test message text goes right here",
							"timestamp": 2
						}
					],
					"logGroupName": "` + cloudWatchConf.LogGroup + `",
					"logStreamName":"` + cloudWatchConf.StreamName + `",
					"sequenceToken":"2"
				}`,
				http.StatusOK,
				`{"nextSequenceToken":"3"}`,
			},
			describeLogStreamsEvent,
			describeLogStreamsEvent,
			describeLogStreamsEvent,
		}

		for _, expect := range expects {
			helpers.GockExpectAPIRequest(t, baseURL, &helpers.APIRequest{
				Method:   expect.ExpectedMethod,
				Body:     expect.ExpectedBody,
				Endpoint: "cloudwatch/",
				ExtraHeaders: http.Header{
					"X-Amz-Target": []string{expect.ExpectedTarget},
				},
			}, &helpers.APIResponse{
				StatusCode: expect.ResultStatusCode,
				Body:       expect.ResultBody,
				Headers: map[string]string{
					"Content-Type": "application/x-amz-json-1.1",
				},
			})
		}

		err := logger.InitZerolog(logger.LoggingConfiguration{
			Debug:                      false,
			LogLevel:                   "debug",
			LoggingToCloudWatchEnabled: true,
		}, cloudWatchConf)
		helpers.FailOnError(t, err)

		log.Error().Msg("test message")
	}, testTimeout)
}
