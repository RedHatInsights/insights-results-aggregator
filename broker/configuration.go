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

// Package broker contains data types, interfaces, and methods related to
// brokers that can be used to consume input messages by aggegator.
package broker

import (
	"time"

	mapset "github.com/deckarep/golang-set"
)

// Configuration represents configuration of Kafka broker
type Configuration struct {
	Address             string        `mapstructure:"address" toml:"address"`
	Topic               string        `mapstructure:"topic" toml:"topic"`
	Timeout             time.Duration `mapstructure:"timeout" toml:"timeout"`
	PayloadTrackerTopic string        `mapstructure:"payload_tracker_topic" toml:"payload_tracker_topic"`
	ServiceName         string        `mapstructure:"service_name" toml:"service_name"`
	Group               string        `mapstructure:"group" toml:"group"`
	Enabled             bool          `mapstructure:"enabled" toml:"enabled"`
	OrgAllowlist        mapset.Set    `mapstructure:"org_whitelist_file" toml:"org_whitelist_file"`
	OrgAllowlistEnabled bool          `mapstructure:"enable_org_whitelist" toml:"enable_org_whitelist"`
}
