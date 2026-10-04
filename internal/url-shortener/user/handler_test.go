package user

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSessionCookieUsesSecureAndMatchingClearPath(t *testing.T) {
	handler := NewUserHandler(nil, 24*time.Hour, time.Hour, true)

	setRecorder := httptest.NewRecorder()
	setContext, _ := gin.CreateTestContext(setRecorder)
	handler.setSessionCookie(setContext, "session", true)
	setCookie := setRecorder.Result().Cookies()[0]

	clearRecorder := httptest.NewRecorder()
	clearContext, _ := gin.CreateTestContext(clearRecorder)
	handler.clearSessionCookie(clearContext)
	clearCookie := clearRecorder.Result().Cookies()[0]

	if !setCookie.Secure || !setCookie.HttpOnly || setCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie security flags are incomplete: %+v", setCookie)
	}
	if setCookie.Path != "/v1/url-shortener" || clearCookie.Path != setCookie.Path || clearCookie.MaxAge >= 0 {
		t.Fatalf("cookie clear does not target the set cookie: set=%+v clear=%+v", setCookie, clearCookie)
	}
}
