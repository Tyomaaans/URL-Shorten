package pkg

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func paginationContext(rawQuery string) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?"+rawQuery, nil)
	return c
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantPage  int
		wantLimit int
		wantErr   bool
	}{
		{name: "defaults", wantPage: 1, wantLimit: 10},
		{name: "valid", query: "page=3&limit=100", wantPage: 3, wantLimit: 100},
		{name: "zero page", query: "page=0", wantErr: true},
		{name: "negative limit", query: "limit=-1", wantErr: true},
		{name: "over maximum", query: "limit=101", wantErr: true},
		{name: "not a number", query: "page=nope", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, limit, err := ParsePagination(paginationContext(tt.query), 10, 100)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && (page != tt.wantPage || limit != tt.wantLimit) {
				t.Fatalf("got page=%d limit=%d, want page=%d limit=%d", page, limit, tt.wantPage, tt.wantLimit)
			}
		})
	}
}
