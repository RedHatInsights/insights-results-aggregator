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

import ()

// Server represents any REST API HTTP server
type Server interface {
	Start() error
}

// Impl in an implementation of Server interface
type Impl struct {
}

// New constructs new implementation of Server interface
func New() Server {
	return Impl{}
}

// Start starts server
func (server Impl) Start() error {
	return nil
}
