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

package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// TestNewRedisClient checks if it is possible to construct Redis client
func TestNewRedisClient(t *testing.T) {
	// default configuration
	configuration := storage.Configuration{
		RedisConfiguration: storage.RedisConfiguration{
			RedisEndpoint:       "localhost:12345",
			RedisDatabase:       0,
			RedisTimeoutSeconds: 1,
			RedisPassword:       "",
		},
	}
	// try to instantiate Redis storage
	client, err := storage.NewRedisStorage(configuration)

	// check results
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

// TestNewDummyRedisClient checks if it is possible to construct Redis
// client structure useful for testing
func TestNewDummyRedisClient(t *testing.T) {
	// configuration where Redis endpoint is set to empty string
	configuration := storage.Configuration{
		RedisConfiguration: storage.RedisConfiguration{
			RedisEndpoint:       "",
			RedisDatabase:       0,
			RedisTimeoutSeconds: 1,
			RedisPassword:       "",
		},
	}
	// try to instantiate Redis storage
	client, err := storage.NewRedisStorage(configuration)

	// check results
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

// TestNewRedisClientDBIndexOutOfRange checks if Redis client
// constructor checks for incorrect database index
func TestNewRedisClientDBIndexOutOfRange(t *testing.T) {
	configuration1 := storage.Configuration{
		RedisConfiguration: storage.RedisConfiguration{
			RedisEndpoint:       "localhost:12345",
			RedisDatabase:       -1,
			RedisTimeoutSeconds: 1,
			RedisPassword:       "",
		},
	}
	// try to instantiate Redis storage
	client, err := storage.NewRedisStorage(configuration1)

	// check results
	assert.Nil(t, client)
	assert.Error(t, err)

	configuration2 := storage.Configuration{
		RedisConfiguration: storage.RedisConfiguration{
			RedisEndpoint:       "localhost:12345",
			RedisDatabase:       16,
			RedisTimeoutSeconds: 1,
			RedisPassword:       "",
		},
	}
	// try to instantiate Redis storage
	client, err = storage.NewRedisStorage(configuration2)

	// check results
	assert.Nil(t, client)
	assert.Error(t, err)
}
