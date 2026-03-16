package minio

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

func TestListObjectVersionsHonorsStartAfter(t *testing.T) {
	startAfter := "b.txt"

	var capturedQuery url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(`<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>false</IsTruncated></ListVersionsResult>`))
	}))
	defer ts.Close()

	srv, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	client, err := New(srv.Host, &Options{
		Creds:  credentials.NewStaticV4("accesskey", "secretkey", ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	for range client.ListObjects(t.Context(), "test-bucket", ListObjectsOptions{
		WithVersions: true,
		StartAfter:   startAfter,
		Recursive:    true,
	}) {
	}

	if capturedQuery.Get("key-marker") != startAfter {
		t.Fatalf("expected key-marker=%q, got %q", startAfter, capturedQuery.Get("key-marker"))
	}
}
