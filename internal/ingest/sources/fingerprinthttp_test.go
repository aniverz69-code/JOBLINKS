package sources

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestFingerprintHTTPBlocksInternalTarget drives the real fingerprintHTTP wrapper (tls-client over
// the SSRF-guarded dialer) at a loopback target and asserts it is refused. This locks the SSRF
// contract through the actual fingerprint transport — the durable protection against a future
// tls-client upgrade silently changing its dial path — and covers the wrapper's error path without
// any real network. No external egress: the guard's Control hook rejects the address before a
// connection is attempted.
func TestFingerprintHTTPBlocksInternalTarget(t *testing.T) {
	c, err := newFingerprintHTTP()
	if err != nil {
		t.Fatalf("newFingerprintHTTP: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var v struct{}
	err = c.GetXML(ctx, "http://127.0.0.1:9/x", &v)
	if err == nil {
		t.Fatal("expected fingerprintHTTP to refuse a loopback target, got nil error")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error %q does not mention the SSRF block", err)
	}
}
