package qwery

import (
	"bytes"
	"context"
	"testing"

	"github.com/redhajuanda/komon/cache"
	"github.com/stretchr/testify/assert"
	// "gitlab.sicepat.tech/platform/qweryalog-sdk-go.git/latency"
	"github.com/redhajuanda/komon/logger"
	gomock "go.uber.org/mock/gomock"
)

func TestCacherTryCache(t *testing.T) {

	var (
		ctx       = context.Background()
		cacheMock = NewMockCache(gomock.NewController(t))
	)
	// ctx = latency.InjectLatencyCounter(ctx)

	t.Run("DoCacheFalse", func(t *testing.T) {
		c := &Cacher{
			doCache: false,
		}

		result := c.tryCache(ctx)

		assert.False(t, result)
	})

	t.Run("ScannerTypeWriter", func(t *testing.T) {

		buf := new(bytes.Buffer)

		c := &Cacher{
			doCache: true,
			cache:   cacheMock,
			log:     logger.New("test"),

			runner: &Runner{
				scanner: &Scanner{
					scannerType: scannerWriter,
					dest:        buf,
				},
			},
		}

		// instrument the cacheMock — Get decodes into *map[string]interface{} for the writer path
		cacheMock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, key string, dest any, _ ...cache.Option) {
				m := dest.(*map[string]interface{})
				*m = map[string]interface{}{"test": "test"}
			},
		).Return(nil)

		result := c.tryCache(ctx)

		assert.True(t, result)
		assert.Equal(t, `{"test":"test"}`, c.runner.scanner.dest.(*bytes.Buffer).String())

	})

	t.Run("ScannerDefault", func(t *testing.T) {

		dest := make(map[string]any)

		c := &Cacher{
			doCache: true,
			cache:   cacheMock,
			log:     logger.New("test"),
			runner: &Runner{
				scanner: &Scanner{
					scannerType: scannerMap,
					dest:        dest,
				},
			},
		}

		// Default path calls Get(ctx, key, *CacheData); mock fills CacheData.Dest for mapper.Decode into scanner dest (map).
		cacheMock.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Do(
			func(ctx context.Context, key string, doc any, _ ...cache.Option) {
				d := doc.(*CacheData)
				d.Dest = map[string]interface{}{"test": "test"}
			},
		).Return(nil)

		result := c.tryCache(ctx)

		assert.True(t, result)
		assert.Equal(t, map[string]any{"test": "test"}, c.runner.scanner.dest.(map[string]any))
	})

}
