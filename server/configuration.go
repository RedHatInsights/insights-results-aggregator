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

package server

// Configuration represents configuration of REST API HTTP server
type Configuration struct {
	Address     string `mapstructure:"address" toml:"address"`
	APIPrefix   string `mapstructure:"api_prefix" toml:"api_prefix"`
	APISpecFile string `mapstructure:"api_spec_file" toml:"api_spec_file"`
	Debug       bool   `mapstructure:"debug" toml:"debug"`
}
