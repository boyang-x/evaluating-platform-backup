package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxExpertDataUploadBytes int64 = 50 << 20

var errUploadTooLarge = errors.New("uploaded file exceeds size limit")

func readLimitedUpload(r io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, errUploadTooLarge
	}
	return data, nil
}

func writeUploadReadError(c *gin.Context, err error) {
	if errors.Is(err, errUploadTooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf("uploaded file is too large; maximum size is %d MB", maxExpertDataUploadBytes>>20),
		})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "read file failed"})
}
