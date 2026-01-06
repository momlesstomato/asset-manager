package checks

import (
	"context"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCheckGameData(t *testing.T) {
	t.Run("GameData All Missing", func(t *testing.T) {
		mockClient := new(MockClient)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

		ch := make(chan minio.ObjectInfo)
		close(ch)
		// For any ListObjects call, return empty channel
		mockClient.On("ListObjects", mock.Anything, "assets", mock.Anything).Return((<-chan minio.ObjectInfo)(ch))

		missing, err := CheckGameData(context.Background(), mockClient, "assets")
		assert.NoError(t, err)
		assert.Len(t, missing, len(RequiredGameDataFiles))
	})

	t.Run("GameData All Present", func(t *testing.T) {
		mockClient := new(MockClient)
		mockClient.On("BucketExists", mock.Anything, "assets").Return(true, nil)

		for _, filename := range RequiredGameDataFiles {
			ch := make(chan minio.ObjectInfo, 1)
			ch <- minio.ObjectInfo{Key: "gamedata/" + filename}
			close(ch)

			targetPrefix := "gamedata/" + filename
			mockClient.On("ListObjects", mock.Anything, "assets", mock.MatchedBy(func(opts minio.ListObjectsOptions) bool {
				return opts.Prefix == targetPrefix
			})).Return((<-chan minio.ObjectInfo)(ch))
		}

		missing, err := CheckGameData(context.Background(), mockClient, "assets")
		assert.NoError(t, err)
		assert.Len(t, missing, 0)
	})
}
