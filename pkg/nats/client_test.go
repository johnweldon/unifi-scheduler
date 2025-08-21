package nats

import (
	"testing"
	"time"
)

func TestClientOptFunctions(t *testing.T) {
	client := &Client{}
	client.Init() // Set defaults

	// Test default values
	if client.connectTimeout != DefaultConnectTimeout {
		t.Errorf("Default connectTimeout = %v, want %v", client.connectTimeout, DefaultConnectTimeout)
	}
	if client.writeTimeout != DefaultWriteTimeout {
		t.Errorf("Default writeTimeout = %v, want %v", client.writeTimeout, DefaultWriteTimeout)
	}
	if client.operationTimeout != DefaultOperationTimeout {
		t.Errorf("Default operationTimeout = %v, want %v", client.operationTimeout, DefaultOperationTimeout)
	}
	if client.streamReplicas != DefaultStreamReplicas {
		t.Errorf("Default streamReplicas = %d, want %d", client.streamReplicas, DefaultStreamReplicas)
	}
	if client.kvReplicas != DefaultKVReplicas {
		t.Errorf("Default kvReplicas = %d, want %d", client.kvReplicas, DefaultKVReplicas)
	}
}

func TestClientOptConfiguration(t *testing.T) {
	testURL := "nats://test:4222"
	testCreds := "/path/to/creds"
	testStreams := []string{"stream1", "stream2"}
	testBuckets := []string{"bucket1", "bucket2"}
	testConnTimeout := 30 * time.Second
	testWriteTimeout := 45 * time.Second
	testOpTimeout := 60 * time.Second
	testStreamReplicas := 5
	testKVReplicas := 7

	client := &Client{}
	client.Init(
		OptNATSUrl(testURL),
		OptCreds(testCreds),
		OptStreams(testStreams...),
		OptBuckets(testBuckets...),
		OptConnectTimeout(testConnTimeout),
		OptWriteTimeout(testWriteTimeout),
		OptOperationTimeout(testOpTimeout),
		OptStreamReplicas(testStreamReplicas),
		OptKVReplicas(testKVReplicas),
	)

	// Test URL setting
	if client.connURL != testURL {
		t.Errorf("connURL = %q, want %q", client.connURL, testURL)
	}

	// Test credentials file setting
	if client.credsFile != testCreds {
		t.Errorf("credsFile = %q, want %q", client.credsFile, testCreds)
	}

	// Test streams setting
	if len(client.streams) != len(testStreams) {
		t.Errorf("len(streams) = %d, want %d", len(client.streams), len(testStreams))
	}
	for i, stream := range testStreams {
		if client.streams[i] != stream {
			t.Errorf("streams[%d] = %q, want %q", i, client.streams[i], stream)
		}
	}

	// Test buckets setting
	if len(client.buckets) != len(testBuckets) {
		t.Errorf("len(buckets) = %d, want %d", len(client.buckets), len(testBuckets))
	}
	for i, bucket := range testBuckets {
		if client.buckets[i] != bucket {
			t.Errorf("buckets[%d] = %q, want %q", i, client.buckets[i], bucket)
		}
	}

	// Test timeout settings
	if client.connectTimeout != testConnTimeout {
		t.Errorf("connectTimeout = %v, want %v", client.connectTimeout, testConnTimeout)
	}
	if client.writeTimeout != testWriteTimeout {
		t.Errorf("writeTimeout = %v, want %v", client.writeTimeout, testWriteTimeout)
	}
	if client.operationTimeout != testOpTimeout {
		t.Errorf("operationTimeout = %v, want %v", client.operationTimeout, testOpTimeout)
	}

	// Test replica settings
	if client.streamReplicas != testStreamReplicas {
		t.Errorf("streamReplicas = %d, want %d", client.streamReplicas, testStreamReplicas)
	}
	if client.kvReplicas != testKVReplicas {
		t.Errorf("kvReplicas = %d, want %d", client.kvReplicas, testKVReplicas)
	}
}

func TestClientMultipleStreamsBuckets(t *testing.T) {
	client := &Client{}

	// Test multiple calls to OptStreams and OptBuckets append
	client.Init(
		OptStreams("stream1", "stream2"),
		OptStreams("stream3"),
		OptBuckets("bucket1"),
		OptBuckets("bucket2", "bucket3"),
	)

	expectedStreams := []string{"stream1", "stream2", "stream3"}
	expectedBuckets := []string{"bucket1", "bucket2", "bucket3"}

	if len(client.streams) != len(expectedStreams) {
		t.Errorf("len(streams) = %d, want %d", len(client.streams), len(expectedStreams))
	}

	if len(client.buckets) != len(expectedBuckets) {
		t.Errorf("len(buckets) = %d, want %d", len(client.buckets), len(expectedBuckets))
	}

	// Verify stream contents
	for i, expected := range expectedStreams {
		if client.streams[i] != expected {
			t.Errorf("streams[%d] = %q, want %q", i, client.streams[i], expected)
		}
	}

	// Verify bucket contents
	for i, expected := range expectedBuckets {
		if client.buckets[i] != expected {
			t.Errorf("buckets[%d] = %q, want %q", i, client.buckets[i], expected)
		}
	}
}

func TestDefaultConstants(t *testing.T) {
	// Test that default constants are reasonable
	if DefaultConnectTimeout <= 0 {
		t.Errorf("DefaultConnectTimeout should be positive, got %v", DefaultConnectTimeout)
	}
	if DefaultWriteTimeout <= 0 {
		t.Errorf("DefaultWriteTimeout should be positive, got %v", DefaultWriteTimeout)
	}
	if DefaultOperationTimeout <= 0 {
		t.Errorf("DefaultOperationTimeout should be positive, got %v", DefaultOperationTimeout)
	}
	if DefaultStreamReplicas <= 0 {
		t.Errorf("DefaultStreamReplicas should be positive, got %d", DefaultStreamReplicas)
	}
	if DefaultKVReplicas <= 0 {
		t.Errorf("DefaultKVReplicas should be positive, got %d", DefaultKVReplicas)
	}

	// Test specific expected values
	expectedConnTimeout := 15 * time.Second
	if DefaultConnectTimeout != expectedConnTimeout {
		t.Errorf("DefaultConnectTimeout = %v, want %v", DefaultConnectTimeout, expectedConnTimeout)
	}

	expectedWriteTimeout := 30 * time.Second
	if DefaultWriteTimeout != expectedWriteTimeout {
		t.Errorf("DefaultWriteTimeout = %v, want %v", DefaultWriteTimeout, expectedWriteTimeout)
	}

	expectedOpTimeout := 30 * time.Second
	if DefaultOperationTimeout != expectedOpTimeout {
		t.Errorf("DefaultOperationTimeout = %v, want %v", DefaultOperationTimeout, expectedOpTimeout)
	}

	if DefaultStreamReplicas != 3 {
		t.Errorf("DefaultStreamReplicas = %d, want 3", DefaultStreamReplicas)
	}

	if DefaultKVReplicas != 3 {
		t.Errorf("DefaultKVReplicas = %d, want 3", DefaultKVReplicas)
	}
}

