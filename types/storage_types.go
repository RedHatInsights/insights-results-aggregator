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

package types

const (
	// SQLStorage constant is used to specify storage based on any SQL database
	SQLStorage = "sql"
	// RedisStorage constant is used to specify storage based on Redis database
	RedisStorage = "redis"
	// NoopStorage constant is used to specify storage based on No-op database
	NoopStorage = "noop"
)
