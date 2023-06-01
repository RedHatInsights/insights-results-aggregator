/*
Copyright © 2020, 2021, 2022, 2023 Red Hat, Inc.

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

package storage

// Configuration represents configuration of data storage
type Configuration struct {
	Driver           string `mapstructure:"db_driver" toml:"db_driver"`
	SQLiteDataSource string `mapstructure:"sqlite_datasource" toml:"sqlite_datasource"`
	LogSQLQueries    bool   `mapstructure:"log_sql_queries" toml:"log_sql_queries"`
	PGUsername       string `mapstructure:"pg_username" toml:"pg_username"`
	PGPassword       string `mapstructure:"pg_password" toml:"pg_password"`
	PGHost           string `mapstructure:"pg_host" toml:"pg_host"`
	PGPort           int    `mapstructure:"pg_port" toml:"pg_port"`
	PGDBName         string `mapstructure:"pg_db_name" toml:"pg_db_name"`
	PGParams         string `mapstructure:"pg_params" toml:"pg_params"`
	Type             string `mapstructure:"type" toml:"type"`
}

// RedisConfiguration represents configuration of Redis client
type RedisConfiguration struct {
	RedisEndpoint       string `mapstructure:"endpoint" toml:"endpoint"`
	RedisDatabase       int    `mapstructure:"database" toml:"database"`
	RedisTimeoutSeconds int    `mapstructure:"timeout_seconds" toml:"timeout_seconds"`
	RedisPassword       string `mapstructure:"password" toml:"password"`
}
