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

package logger

// LoggingConfiguration represents configuration for logging in general
type LoggingConfiguration struct {
	// Debug enables pretty colored logging
	Debug bool `mapstructure:"debug" toml:"debug"`

	// LoggingToCloudWatchEnabled enables logging to CloudWatch
	// (configuration for CloudWatch is in CloudWatchConfiguration)
	LoggingToCloudWatchEnabled bool `mapstructure:"logging_to_cloud_watch_enabled" toml:"logging_to_cloud_watch_enabled"`
}

// CloudWatchConfiguration represents configuration of CloudWatch logger
type CloudWatchConfiguration struct {
	AWSAccessID     string `mapstructure:"aws_access_id" toml:"aws_access_id"`
	AWSSecretKey    string `mapstructure:"aws_secret_key" toml:"aws_secret_key"`
	AWSSessionToken string `mapstructure:"aws_session_token" toml:"aws_session_token"`
	AWSRegion       string `mapstructure:"aws_region" toml:"aws_region"`
	LogGroup        string `mapstructure:"log_group" toml:"log_group"`
	StreamName      string `mapstructure:"stream_name" toml:"stream_name"`

	// enable debug logs for debugging aws client itself
	Debug bool `mapstructure:"debug" toml:"debug"`
}
