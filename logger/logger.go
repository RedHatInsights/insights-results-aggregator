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
	"io"
	"os"

	cloudwatch "github.com/Sergey1011010/ejholmes-cloudwatch"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitZerolog initializes zerolog with provided configs to use proper stdout and/or CloudWatch logging
func InitZerolog(loggingConf LoggingConfiguration, cloudWatchConf CloudWatchConfiguration) error {
	var writers []io.Writer

	if loggingConf.Debug {
		// nice colored output
		writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout})
	} else {
		writers = append(writers, os.Stdout)
	}

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
		group := cloudwatch.NewGroup(cloudWatchConf.LogGroup, cloudwatchlogs.New(cloudWatchSession))

		cloudWatchWriter, err := group.Create(cloudWatchConf.StreamName)
		if err != nil {
			return err
		}

		writers = append(writers, cloudWatchWriter)
	}

	logsWriter := io.MultiWriter(writers...)

	log.Logger = zerolog.New(logsWriter).With().Timestamp().Logger()

	return nil
}
