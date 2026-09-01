package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestAttackSampleUploadReturnsUnavailableWhenStorageIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("sub_type=compliance&name=demo"))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	NewAttackSampleHandler(nil, nil, nil, nil).Upload(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
}

func TestAttackSamplePreviewReturnsUnavailableWhenStorageIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Request = httptest.NewRequest(http.MethodGet, "/?limit=1", nil)

	NewAttackSampleHandler(nil, nil, nil, nil).Preview(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
}

func TestComposedAttackUploadReturnsUnavailableWhenStorageIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=demo"))
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	NewComposedAttackHandler(nil, nil, nil, nil).Upload(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
}

func TestComposedAttackPreviewReturnsUnavailableWhenStorageIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: uuid.NewString()}}
	c.Request = httptest.NewRequest(http.MethodGet, "/?limit=1", nil)

	NewComposedAttackHandler(nil, nil, nil, nil).Preview(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
}
