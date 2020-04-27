package consumer_test

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
)

func benchmarkProcessingMessage(b *testing.B, s storage.Storage, messageProducer func() string) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	kafkaConsumer := &consumer.KafkaConsumer{
		Storage: s,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mustConsumerProcessMessage(b, kafkaConsumer, messageProducer())
	}
}

func getNoopStorage(b *testing.B) (storage.Storage, func(*testing.B)) {
	return &storage.NoopStorage{}, nil
}

func getSQLiteMemoryStorage(b *testing.B) (storage.Storage, func(*testing.B)) {
	return helpers.MustGetMockStorage(b, true), nil
}

func getSQLiteFileStorage(b *testing.B) (storage.Storage, func(*testing.B)) {
	const dbFile = "./test.db"

	db, err := sql.Open("sqlite3", dbFile)
	helpers.FailOnError(b, err)

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	helpers.FailOnError(b, err)

	sqliteStorage := storage.NewFromConnection(db, storage.DBDriverSQLite3)

	err = sqliteStorage.Init()
	helpers.FailOnError(b, err)

	return sqliteStorage, func(b *testing.B) {
		os.Remove(dbFile)
	}
}

func getPostgresStorage(b *testing.B) (storage.Storage, func(*testing.B)) {
	err := conf.LoadConfiguration("../config-devel")
	helpers.FailOnError(b, err)

	postgresStorage, err := storage.New(conf.GetStorageConfiguration())
	helpers.FailOnError(b, err)

	return postgresStorage, nil
}

func BenchmarkKafkaConsumer_ProcessMessage_SimpleMessages(b *testing.B) {
	var testCases = []struct {
		Name            string
		StorageProducer func(*testing.B) (storage.Storage, func(*testing.B))
		RandomMessages  bool
	}{
		{"NoopStorage", getNoopStorage, false},
		{"NoopStorage", getNoopStorage, true},
		{"Postgres", getPostgresStorage, false},
		{"Postgres", getPostgresStorage, true},
		{"SQLiteInMemory", getSQLiteMemoryStorage, false},
		{"SQLiteInMemory", getSQLiteMemoryStorage, true},
		{"SQLiteFile", getSQLiteFileStorage, false},
		{"SQLiteFile", getSQLiteFileStorage, true},
	}

	for _, testCase := range testCases {
		if testCase.RandomMessages {
			testCase.Name += "/RandomMessages"
		}

		b.Run(testCase.Name, func(b *testing.B) {
			benchStorage, cleaner := testCase.StorageProducer(b)
			if cleaner != nil {
				defer cleaner(b)
			}
			defer helpers.MustCloseStorage(b, benchStorage)

			if testCase.RandomMessages {
				benchmarkProcessingMessage(b, benchStorage, func() string {
					return testdata.GetRandomConsumerMessage()
				})
			} else {
				benchmarkProcessingMessage(b, benchStorage, func() string {
					return testdata.ConsumerMessage
				})
			}
		})
	}
}
