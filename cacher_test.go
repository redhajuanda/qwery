package qwery

import (
	"bytes"
	"context"
	"testing"

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

		// instrument the cacheMock
		cacheMock.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]byte(`{"test":"test"}`), nil)

		result := c.tryCache(ctx)

		assert.True(t, result)
		assert.Equal(t, `{"test":"test"}`, c.runner.scanner.dest.(*bytes.Buffer).String())

	})

	t.Run("ScannerDefault", func(t *testing.T) {

		var (
			dest = CacheData{}
		)

		c := &Cacher{
			doCache: true,
			cache:   cacheMock,
			log:     logger.New("test"),
			runner: &Runner{
				// result: &result.Result{
				// 	Metadata: &result.Metadata{
				// 		HTTP: &result.HTTPData{},
				// 	},
				// },
				scanner: &Scanner{
					scannerType: scannerMap,
					dest:        &dest,
				},
			},
		}

		// instrument the cacheMock
		cacheMock.EXPECT().
			GetObject(gomock.Any(), gomock.Any(), &dest).
			Do(func(ctx context.Context, key string, doc interface{}) error {

				// set the dest
				dest.Dest = map[string]interface{}{"test": "test"}
				return nil
			}).
			Return(nil)

		result := c.tryCache(ctx)

		assert.True(t, result)
		assert.Equal(t, map[string]interface{}{"test": "test"}, c.runner.scanner.dest.(*CacheData).Dest)

	})

}
