/*
Copyright © 2019, 2020 Red Hat, Inc.

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

package main

// Export for testing
//
// This source file contains name aliases of all package-private functions
// that need to be called from unit tests. Aliases should start with uppercase
// letter because unit tests belong to different package.
//
// Please look into the following blogpost:
// https://medium.com/@robiplus/golang-trick-export-for-test-aa16cbd7b8cd
// to see why this trick is needed.
var (
	CreateStorage       = createStorage
	StartService        = startService
	StopService         = stopService
	CloseStorage        = closeStorage
	PrepareDB           = prepareDB
	StartConsumer       = startConsumer
	StartServer         = startServer
	PrintVersionInfo    = printVersionInfo
	PrintHelp           = printHelp
	PrintConfig         = printConfig
	PrintEnv            = printEnv
	GetDBForMigrations  = getDBForMigrations
	PrintMigrationInfo  = printMigrationInfo
	SetMigrationVersion = setMigrationVersion
	PerformMigrations   = performMigrations
	AutoMigratePtr      = &autoMigrate
	Main                = main
)
