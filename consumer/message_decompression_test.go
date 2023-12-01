/*
Copyright © 2023 Red Hat, Inc.

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

package consumer_test

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/stretchr/testify/assert"
)

func TestIsMessageInGzipFormatForEmptyMessage(t *testing.T) {
	message := []byte{}
	result := consumer.IsMessageInGzipFormat(message)
	assert.False(t, result, "Improper message format detection for empty message")
}

func TestIsMessageInGzipFormatForShortMessage(t *testing.T) {
	message := []byte{31}
	result := consumer.IsMessageInGzipFormat(message)
	assert.False(t, result, "Improper message format detection for too short message")
}

func TestIsMessageInGzipFormatForMessageWithExpectedHeader(t *testing.T) {
	message := []byte{31, 139}
	result := consumer.IsMessageInGzipFormat(message)
	assert.True(t, result, "Improper message format detection for message with right header")
}

func TestIsMessageInGzipFormatForCompressedMessage(t *testing.T) {
	original := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	compressed := compressConsumerMessage(original)

	result := consumer.IsMessageInGzipFormat(compressed)
	assert.True(t, result, "Improper message format detection for compressed message")
}

func TestDecompressEmptyMessage(t *testing.T) {
	message := []byte{}
	decompressed, err := consumer.DecompressMessage(message)
	assert.NoError(t, err, "No error is expected")
	assert.Equal(t, message, decompressed, "Message should not be decompressed")
}

func TestDecompressNonCompressedMessage(t *testing.T) {
	message := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	decompressed, err := consumer.DecompressMessage(message)
	assert.NoError(t, err, "No error is expected")
	assert.Equal(t, message, decompressed, "Message should not be decompressed")
}

func compressConsumerMessage(original []byte) []byte {
	compressed := new(bytes.Buffer)
	gzipWritter := gzip.NewWriter(compressed)
	_, err := gzipWritter.Write(original)
	if err != nil {
		panic(err)
	}
	err = gzipWritter.Flush()
	if err {
		panic(err)
	}
	err = gzipWritter.Close()
	if err {
		panic(err)
	}
	return compressed.Bytes()
}

func TestDecompressCompressedMessage(t *testing.T) {
	original := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	compressed := compressConsumerMessage(original)

	decompressed, err := consumer.DecompressMessage(compressed)
	assert.NoError(t, err, "No error is expected")
	assert.Equal(t, original, decompressed, "Message is not decompressed properly")
}
