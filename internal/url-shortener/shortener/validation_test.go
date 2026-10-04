package shortener

import (
	"testing"

	"url-shorten/pkg"
)

func TestShortenRequestsOnlyAcceptAbsoluteHTTPURLs(t *testing.T) {
	validate := pkg.NewValidator()
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "https", value: "https://example.com/path?q=1"},
		{name: "http", value: "http://example.com"},
		{name: "empty", value: "", wantErr: true},
		{name: "relative", value: "/internal/path", wantErr: true},
		{name: "javascript", value: "javascript:alert(1)", wantErr: true},
		{name: "data", value: "data:text/html,hello", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pkg.ValidateStruct(validate, CreateShortenPublicRequest{OriginalURL: tt.value})
			if (err != nil) != tt.wantErr {
				t.Fatalf("validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthorizedShortenUsesTheSameURLPolicy(t *testing.T) {
	validate := pkg.NewValidator()
	bad := "file:///etc/passwd"
	if err := pkg.ValidateStruct(validate, CreateShortenAuthorizedRequest{OriginalURL: bad}); err == nil {
		t.Fatal("authorized create accepted a non-HTTP URL")
	}
	if err := pkg.ValidateStruct(validate, UpdateShortenAuthorizedRequest{OriginalURL: &bad}); err == nil {
		t.Fatal("authorized update accepted a non-HTTP URL")
	}
}
