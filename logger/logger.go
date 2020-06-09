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

// Package logger contains the configuration structures needed to configure
// the access to CloudWatch server to sending the log messages there
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/RedHatInsights/cloudwatch"
	"github.com/Shopify/sarama"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// UnJSONWriter converts JSON objects to not JSON to fix RHIOPS-729.
// TODO: delete when RHIOPS-729 is fixed
type UnJSONWriter struct {
	io.Writer
}

func (writer UnJSONWriter) Write(bytes []byte) (int, error) {
	var obj map[string]interface{}

	err := json.Unmarshal(bytes, &obj)
	if err != nil {
		// it's not JSON object, so we don't do anything
		return writer.Writer.Write(bytes)
	}

	// uppercase the keys
	for key := range obj {
		val := obj[key]
		delete(obj, key)
		obj[strings.ToUpper(key)] = val
	}

	stringifiedObj := ""

	processKeyIfExists := func(key string) {
		if val, ok := obj[key]; ok {
			stringifiedObj += fmt.Sprintf("%+v=%+v; ", strings.ToUpper(key), val)
			delete(obj, key)
		}
	}

	processKeyIfExists("LEVEL")
	processKeyIfExists("TIME")
	processKeyIfExists("ERROR")
	processKeyIfExists("MESSAGE")

	// process the rest
	for key, val := range obj {
		stringifiedObj += fmt.Sprintf("%+v=%+v; ", strings.ToUpper(key), val)
	}

	written, err := writer.Write([]byte(stringifiedObj))
	if err != nil {
		return written, err
	}

	if written < len(stringifiedObj) {
		return written, fmt.Errorf("too few bytes were written")
	}

	return len(bytes), nil
}

// InitZerolog initializes zerolog with provided configs to use proper stdout and/or CloudWatch logging
func InitZerolog(
	loggingConf LoggingConfiguration, cloudWatchConf CloudWatchConfiguration, additionalWriters ...io.Writer,
) error {
	setGlobalLogLevel(loggingConf)

	var writers []io.Writer

	writers = append(writers, additionalWriters...)

	if loggingConf.Debug {
		// nice colored output
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout})
	} else {
		writers = append(writers, os.Stdout)
	}

	cloudWatchConf.StreamName = strings.ReplaceAll(cloudWatchConf.StreamName, "$HOSTNAME", os.Getenv("HOSTNAME"))

	if loggingConf.LoggingToCloudWatchEnabled {
		awsLogLevel := aws.LogOff
		if cloudWatchConf.Debug {
			awsLogLevel = aws.LogDebugWithSigning |
				aws.LogDebugWithSigning |
				aws.LogDebugWithHTTPBody |
				aws.LogDebugWithEventStreamBody
		}

		awsConf := aws.NewConfig().
			WithCredentials(credentials.NewStaticCredentials(
				cloudWatchConf.AWSAccessID, cloudWatchConf.AWSSecretKey, cloudWatchConf.AWSSessionToken,
			)).
			WithRegion(cloudWatchConf.AWSRegion).
			WithLogLevel(awsLogLevel)

		cloudWatchSession := session.Must(session.NewSession(awsConf))
		cloudWatchClient := cloudwatchlogs.New(cloudWatchSession)

		var cloudWatchWriter io.Writer
		if cloudWatchConf.CreateStreamIfNotExists {
			group := cloudwatch.NewGroup(cloudWatchConf.LogGroup, cloudWatchClient)

			var err error
			cloudWatchWriter, err = group.Create(cloudWatchConf.StreamName)
			if err != nil {
				return err
			}
		} else {
			cloudWatchWriter = cloudwatch.NewWriter(cloudWatchConf.LogGroup, cloudWatchConf.StreamName, cloudWatchClient)
		}

		writers = append(writers, &UnJSONWriter{Writer: cloudWatchWriter})
	}

	logsWriter := io.MultiWriter(writers...)

	log.Logger = zerolog.New(logsWriter).With().Timestamp().Logger()

	// zerolog doesn't implement Println required by sarama
	sarama.Logger = &SaramaZerologger{zerologger: log.Logger}

	return nil
}

func setGlobalLogLevel(configuration LoggingConfiguration) {
	logLevel := strings.ToLower(strings.TrimSpace(configuration.LogLevel))

	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	}
}

const kafkaErrorPrefix = "kafka: error"

// SaramaZerologger is a wrapper to make sarama log to zerolog
// those logs can be filtered by key "package" with value "sarama"
type SaramaZerologger struct{ zerologger zerolog.Logger }

// Print wraps print method
func (logger *SaramaZerologger) Print(params ...interface{}) {
	var messages []string
	for _, item := range params {
		messages = append(messages, fmt.Sprint(item))
	}

	logger.logMessage("%v", strings.Join(messages, " "))
}

// Printf wraps printf method
func (logger *SaramaZerologger) Printf(format string, params ...interface{}) {
	logger.logMessage(format, params...)
}

// Println wraps println method
func (logger *SaramaZerologger) Println(v ...interface{}) {
	logger.Print(v...)
}

func (logger *SaramaZerologger) logMessage(format string, params ...interface{}) {
	var event *zerolog.Event
	messageStr := fmt.Sprintf(format, params...)

	if strings.HasPrefix(messageStr, kafkaErrorPrefix) {
		event = logger.zerologger.Error()
	} else {
		event = logger.zerologger.Info()
	}

	event = event.Str("package", "sarama")
	event.Msg(messageStr)
}
