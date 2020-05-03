package youtube

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rylio/ytdl"
	"github.com/schollz/logger"
	"github.com/stretchr/testify/assert"
)

func TestDownload(t *testing.T) {
	logger.SetLevel("trace")
	assert.Nil(t, Download("https://www.youtube.com/watch?v=aY6FW9teGNQ"))
	assert.Nil(t, Download("aY6FW9teGNQ"))
}

func Download(u string) (err error) {
	log.Logger = log.Output(os.Stderr)
	log.Logger = log.Level(zerolog.FatalLevel)
	client := ytdl.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Logger,
	}
	info, err := client.GetVideoInfo(context.Background(), u)
	if err != nil {
		logger.Debug(err)
		return
	}
	logger.Debugf("info: %+v", info)

	return
}
