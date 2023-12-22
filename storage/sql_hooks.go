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

package storage

import (
	"context"
	"database/sql"
	sql_driver "database/sql/driver"
	"encoding/json"
	"time"

	"github.com/gchaincl/sqlhooks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
)

type sqlHooks struct{}

type sqlHooksKey int

const (
	sqlHooksKeyQueryBeginTime sqlHooksKey = iota
)

// LogFormatterString is format string for sql queries logging
// first arg is query string
// second arg is params array
const logFormatterString = "query `%+v` with params `%+v`"

// Before is called before the query was executed allowing yout to log what you asked db to do
func (h *sqlHooks) Before(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	jsonArgs, err := json.Marshal(args)
	if err == nil {
		h.log(logFormatterString+"\n", query, string(jsonArgs))
	} else {
		h.log(logFormatterString+"\n", query, args)
	}

	metrics.SQLQueriesCounter.Inc()

	return context.WithValue(ctx, sqlHooksKeyQueryBeginTime, time.Now()), nil
}

// After is called after the query was executed showing only successful ones
// it allows you to see how long your query took
func (h *sqlHooks) After(ctx context.Context, query string, args ...interface{}) (context.Context, error) {
	beginTime := ctx.Value(sqlHooksKeyQueryBeginTime).(time.Time)
	duration := time.Since(beginTime)

	metrics.SQLQueriesDurations.With(prometheus.Labels{"query": query}).Observe(duration.Seconds())

	jsonArgs, err := json.Marshal(args)
	if err == nil {
		h.log(
			logFormatterString+" took %s\n",
			query, string(jsonArgs), duration,
		)
	} else {
		h.log(
			logFormatterString+" took %s\n",
			query, args, duration,
		)
	}

	return ctx, nil
}

func (h *sqlHooks) log(format string, params ...interface{}) {
	log.Debug().Str("type", "SQL").Msgf(format, params...)
}

// InitSQLDriverWithLogs initializes wrapped version of driver with logging sql queries
// and returns its name
func InitSQLDriverWithLogs(
	realDriver sql_driver.Driver,
	realDriverName string,
) string {
	// linear search is not going to be an issue since there's not many drivers
	// and we call New() only ones/twice per process life
	foundHooksDriver := false
	hooksDriverName := realDriverName + "WithHooks"

	for _, existingDriver := range sql.Drivers() {
		if existingDriver == hooksDriverName {
			foundHooksDriver = true
			break
		}
	}

	if !foundHooksDriver {
		sql.Register(hooksDriverName, sqlhooks.Wrap(realDriver, &sqlHooks{}))
	}

	return hooksDriverName
}
