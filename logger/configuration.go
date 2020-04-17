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

// Configuration for CloudWatch logger
package logger

// Configuration represents configuration of CloudWatch logger
type Configuration struct {
	AWS_access_id  string `mapstructure:"aws_access_id" toml:"aws_access_id"`
	AWS_secret_key string `mapstructure:"aws_secret_key" toml:"aws_secret_key"`
	AWS_region     string `mapstructure:"aws_region" toml:"aws_region"`
	LogGroup       string `mapstructure:"log_group" toml:"log_group"`
	StreamName     string `mapstructure:"stream_name" toml:"stream_name"`
}
