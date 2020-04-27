package consumer_test

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/conf"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
)

func benchmarkProcessingMessage(b *testing.B, s storage.Storage, messageProducer func() string) {
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
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	var testCases = []struct {
		Name            string
		StorageProducer func(*testing.B) (storage.Storage, func(*testing.B))
		RandomMessages  bool
	}{
		{"NoopStorage", getNoopStorage, false},
		{"NoopStorage", getNoopStorage, true},
		{"SQLiteInMemory", getSQLiteMemoryStorage, false},
		{"SQLiteInMemory", getSQLiteMemoryStorage, true},
		{"Postgres", getPostgresStorage, false},
		{"Postgres", getPostgresStorage, true},
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

func BenchmarkKafkaConsumer_ProcessMessage_RealMessages(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	const dataDir = "../utils/produce_insights_results/"
	files, err := ioutil.ReadDir(dataDir)
	helpers.FailOnError(b, err)

	var messages []string

	for _, file := range files {
		if file.Mode().IsRegular() {
			if strings.HasSuffix(file.Name(), ".json") && !strings.Contains(file.Name(), "broken") {
				filePath := path.Join(dataDir, file.Name())

				fileBytes, err := ioutil.ReadFile(filePath)
				helpers.FailOnError(b, err)

				zerolog.SetGlobalLevel(zerolog.Disabled)
				parsedMessage, err := consumer.ParseMessage(fileBytes)
				zerolog.SetGlobalLevel(zerolog.WarnLevel)
				if err != nil {
					log.Warn().Msgf("skipping file %+v because it has bad structure", file.Name())
					continue
				}
				err = consumer.CheckReportStructure(*parsedMessage.Report)
				if err != nil {
					log.Warn().Msgf("skipping file %+v because its report has bad structure", file.Name())
					continue
				}

				fileContent := string(fileBytes)

				messages = append(messages, fileContent)
			}
		}
	}

	var testCases = []struct {
		Name            string
		StorageProducer func(*testing.B) (storage.Storage, func(*testing.B))
	}{
		{"NoopStorage", getNoopStorage},
		{"SQLiteInMemory", getSQLiteMemoryStorage},
		{"Postgres", getPostgresStorage},
		{"SQLiteFile", getSQLiteFileStorage},
	}

	for _, testCase := range testCases {
		testCase.Name += "/" + testCase.Name

		b.Run(testCase.Name, func(b *testing.B) {
			benchStorage, cleaner := testCase.StorageProducer(b)
			if cleaner != nil {
				defer cleaner(b)
			}
			defer helpers.MustCloseStorage(b, benchStorage)

			kafkaConsumer := &consumer.KafkaConsumer{
				Storage: benchStorage,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, message := range messages {
					mustConsumerProcessMessage(b, kafkaConsumer, message)
				}
			}
		})
	}
}
