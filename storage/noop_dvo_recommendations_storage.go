// Copyright 2023 Red Hat, Inc
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

package storage

import "database/sql"

// NoopDVOStorage represents a storage which does nothing (for benchmarking without a storage)
type NoopDVOStorage struct{}

// Init noop
func (*NoopDVOStorage) Init() error {
	return nil
}

// Close noop
func (*NoopDVOStorage) Close() error {
	return nil
}

// GetConnection noop
func (*NoopDVOStorage) GetConnection() *sql.DB {
	return nil
}
