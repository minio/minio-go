package minio

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TestComposeObjectPinsSourceETag verifies that the multipart-copy path of
// ComposeObject sends x-amz-copy-source-if-match for every source, pinned to
// the ETag observed by the StatObject it performs to size the parts.
func TestComposeObjectPinsSourceETag(t *testing.T) {
	const srcSize = 5 * 1024 * 1024 // one part per source at the default part size

	var (
		mu      sync.Mutex
		gotETag []string
		gotPath []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead:
			// StatObject for a compose source.
			name := strings.TrimPrefix(r.URL.Path, "/test-bucket/")
			w.Header().Set("ETag", `"etag-`+name+`"`)
			w.Header().Set("Content-Length", strconv.Itoa(srcSize))
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Query().Has("uploads"):
			io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>test-bucket</Bucket><Key>dst</Key><UploadId>upload-1</UploadId></InitiateMultipartUploadResult>`)
		case r.Method == http.MethodPut && r.URL.Query().Get("partNumber") != "":
			mu.Lock()
			gotETag = append(gotETag, r.Header.Get("x-amz-copy-source-if-match"))
			gotPath = append(gotPath, r.Header.Get("x-amz-copy-source"))
			mu.Unlock()
			fmt.Fprintf(w, `<CopyPartResult><ETag>"part-%s"</ETag><LastModified>2020-01-01T00:00:00.000Z</LastModified></CopyPartResult>`, r.URL.Query().Get("partNumber"))
		case r.Method == http.MethodPost && r.URL.Query().Get("uploadId") != "":
			io.WriteString(w, `<CompleteMultipartUploadResult><Bucket>test-bucket</Bucket><Key>dst</Key><ETag>"final"</ETag></CompleteMultipartUploadResult>`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	clnt, err := New(strings.TrimPrefix(ts.URL, "http://"), &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = clnt.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "test-bucket", Object: "dst"},
		CopySrcOptions{Bucket: "test-bucket", Object: "a"},
		CopySrcOptions{Bucket: "test-bucket", Object: "b"},
	)
	if err != nil {
		t.Fatalf("ComposeObject: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(gotPath) != 2 {
		t.Fatalf("got %d upload-part-copy requests, want 2", len(gotPath))
	}
	want := []string{"etag-a", "etag-b"}
	for i := range want {
		if gotETag[i] != want[i] {
			t.Errorf("upload-part-copy %d (source %q): x-amz-copy-source-if-match = %q, want %q",
				i+1, gotPath[i], gotETag[i], want[i])
		}
	}
}

// A caller-supplied MatchETag that disagrees with the ETag observed while
// sizing the parts cannot describe the version the ranges were computed from.
func TestComposeObjectRejectsMismatchedMatchETag(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			name := strings.TrimPrefix(r.URL.Path, "/test-bucket/")
			w.Header().Set("ETag", `"etag-`+name+`"`)
			w.Header().Set("Content-Length", strconv.Itoa(5*1024*1024))
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
			return
		}
		t.Errorf("unexpected request after a mismatched MatchETag: %s %s", r.Method, r.URL)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	clnt, err := New(strings.TrimPrefix(ts.URL, "http://"), &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = clnt.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "test-bucket", Object: "dst"},
		CopySrcOptions{Bucket: "test-bucket", Object: "a", MatchETag: "etag-other"},
		CopySrcOptions{Bucket: "test-bucket", Object: "b"},
	)
	if err == nil {
		t.Fatal("ComposeObject accepted a MatchETag that differs from the statted ETag")
	}
	if !strings.Contains(err.Error(), "does not match the ETag") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// A caller-supplied MatchETag that agrees with the statted source must be kept
// verbatim (including its quotes), not replaced by the statted value.
func TestComposeObjectKeepsMatchingMatchETag(t *testing.T) {
	const srcSize = 5 * 1024 * 1024
	var (
		mu   sync.Mutex
		etag []string
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead:
			name := strings.TrimPrefix(r.URL.Path, "/test-bucket/")
			w.Header().Set("ETag", `"etag-`+name+`"`)
			w.Header().Set("Content-Length", strconv.Itoa(srcSize))
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Query().Has("uploads"):
			io.WriteString(w, `<InitiateMultipartUploadResult><Bucket>test-bucket</Bucket><Key>dst</Key><UploadId>upload-1</UploadId></InitiateMultipartUploadResult>`)
		case r.Method == http.MethodPut && r.URL.Query().Get("partNumber") != "":
			mu.Lock()
			etag = append(etag, r.Header.Get("x-amz-copy-source-if-match"))
			mu.Unlock()
			fmt.Fprintf(w, `<CopyPartResult><ETag>"part-%s"</ETag><LastModified>2020-01-01T00:00:00.000Z</LastModified></CopyPartResult>`, r.URL.Query().Get("partNumber"))
		case r.Method == http.MethodPost && r.URL.Query().Get("uploadId") != "":
			io.WriteString(w, `<CompleteMultipartUploadResult><Bucket>test-bucket</Bucket><Key>dst</Key><ETag>"final"</ETag></CompleteMultipartUploadResult>`)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	clnt, err := New(strings.TrimPrefix(ts.URL, "http://"), &Options{
		Creds:  credentials.NewStaticV4("ak", "sk", ""),
		Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = clnt.ComposeObject(context.Background(),
		CopyDestOptions{Bucket: "test-bucket", Object: "dst"},
		CopySrcOptions{Bucket: "test-bucket", Object: "a", MatchETag: `"etag-a"`},
		CopySrcOptions{Bucket: "test-bucket", Object: "b"},
	)
	if err != nil {
		t.Fatalf("ComposeObject: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	want := []string{`"etag-a"`, "etag-b"}
	if len(etag) != len(want) {
		t.Fatalf("got %d upload-part-copy requests, want %d", len(etag), len(want))
	}
	for i := range want {
		if etag[i] != want[i] {
			t.Errorf("upload-part-copy %d: x-amz-copy-source-if-match = %q, want %q", i+1, etag[i], want[i])
		}
	}
}
