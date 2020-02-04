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

package consumer

import (
	"github.com/RedHatInsights/insights-results-aggregator/broker"
)

// Consumer represents any consumer of insights-rules messages
type Consumer interface {
	Start() error
}

// Impl in an implementation of Consumer interface
type Impl struct {
}

// New constructs new implementation of Consumer interface
func New(brokerCfg broker.Configuration) Consumer {
	return Impl{}
}

// Start starts consumer
func (consumer Impl) Start() error {
	return nil
}
