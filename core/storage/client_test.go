package storage_test

import (
	"testing"

	"asset-manager/core/storage"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := storage.Config{
			Endpoint:  "localhost:9000",
			AccessKey: "testkey",
			SecretKey: "testsecret",
			UseSSL:    false,
			Bucket:    "test-bucket",
			Region:    "us-east-1",
		}

		client, err := storage.NewClient(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("EndpointWithHTTP", func(t *testing.T) {
		cfg := storage.Config{
			Endpoint:  "http://localhost:9000",
			AccessKey: "testkey",
			SecretKey: "testsecret",
			UseSSL:    false,
		}

		client, err := storage.NewClient(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("EndpointWithHTTPS", func(t *testing.T) {
		cfg := storage.Config{
			Endpoint:  "https://s3.amazonaws.com",
			AccessKey: "testkey",
			SecretKey: "testsecret",
			UseSSL:    true,
			Region:    "us-east-1",
		}

		client, err := storage.NewClient(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, client)
	})
}
