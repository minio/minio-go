package minio

import (
	"net/http"
	"testing"
)

func TestSuccessStatusIncludesAccepted(t *testing.T) {
	if !successStatus.Contains(http.StatusAccepted) {
		t.Fatal("expected 202 Accepted to be treated as a successful response")
	}
}
