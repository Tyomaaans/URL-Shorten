package router

import (
	"testing"

	"url-shorten/internal/middleware"
)

func TestPublicShortenCreatePolicies(t *testing.T) {
	policies := publicShortenCreatePolicies()
	if len(policies) != 2 {
		t.Fatalf("policy count = %d, want 2", len(policies))
	}
	if policies[0].Identity != middleware.IdentityIP || policies[0].Limit != 25 {
		t.Fatalf("unexpected IP policy: %+v", policies[0])
	}
	if policies[1].Identity != middleware.IdentityIPUserAgent || policies[1].Limit != 5 {
		t.Fatalf("unexpected client policy: %+v", policies[1])
	}
}
