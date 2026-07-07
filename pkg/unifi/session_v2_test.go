package unifi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestSession(t *testing.T, handler http.Handler) (*Session, *httptest.Server) {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	creds, err := NewCredentials("testuser", "testpass")
	if err != nil {
		t.Fatalf("creating credentials: %v", err)
	}

	session := &Session{Endpoint: server.URL}
	if err := session.Initialize(
		WithCredentials(creds),
		WithInsecureTLS(),
		WithHTTPTimeout(5*time.Second),
	); err != nil {
		t.Fatalf("initializing session: %v", err)
	}

	return session, server
}

func notFoundJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `{"meta":{"rc":"error","msg":"api.err.NotFound"}}`)
}

func okDataJSON(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"meta":{"rc":"ok"},"data":%s}`, data)
}

// TestSession_GetRecentEvents_UsesV2SystemLog verifies the events fetch uses
// the v2 system-log API (POST /proxy/network/v2/api/site/{site}/system-log/all)
// instead of the removed legacy /stat/event endpoint.
func TestSession_GetRecentEvents_UsesV2SystemLog(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody SystemLogRequest

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/stat/event") || strings.Contains(r.URL.Path, "/rest/event") {
			notFoundJSON(w)
			return
		}

		if strings.Contains(r.URL.Path, "/system-log/") {
			gotPath = r.URL.Path
			gotMethod = r.Method

			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("decoding system-log request body: %v", err)
			}

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{
				"data": [
					{
						"id": "evt-2",
						"key": "CLIENT_CONNECTED",
						"category": "CLIENT_DEVICES",
						"message_raw": "{CLIENT} connected to the network",
						"parameters": {"CLIENT": {"id": "aa:bb:cc:dd:ee:ff", "name": "laptop", "hostname": "laptop.local"}},
						"timestamp": 1751700060000
					},
					{
						"id": "evt-1",
						"key": "CLIENT_DISCONNECTED",
						"category": "CLIENT_DEVICES",
						"message_raw": "{CLIENT} disconnected from the network",
						"parameters": {"CLIENT": {"id": "aa:bb:cc:dd:ee:ff", "name": "laptop", "hostname": "laptop.local"}},
						"timestamp": 1751700000000
					}
				],
				"page_number": 0,
				"total_element_count": 2,
				"total_page_count": 1
			}`)
			return
		}

		notFoundJSON(w)
	})

	session, _ := newTestSession(t, handler)

	events, err := session.GetRecentEvents()
	if err != nil {
		t.Fatalf("GetRecentEvents failed: %v", err)
	}

	if want := "/proxy/network/v2/api/site/default/system-log/all"; gotPath != want {
		t.Errorf("expected path %q, got %q", want, gotPath)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %q", gotMethod)
	}

	if len(gotBody.Categories) == 0 {
		t.Error("expected request body to include categories")
	}

	if gotBody.PageSize == 0 {
		t.Error("expected request body to include a page size")
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Sorted oldest first (DefaultEventSort by DateTime).
	if events[0].ID != "evt-1" || events[1].ID != "evt-2" {
		t.Errorf("expected events sorted by time [evt-1 evt-2], got [%s %s]", events[0].ID, events[1].ID)
	}

	first := events[0]

	if first.Key != EventType("CLIENT_DISCONNECTED") {
		t.Errorf("expected key CLIENT_DISCONNECTED, got %q", first.Key)
	}

	if want := "laptop disconnected from the network"; first.Message != want {
		t.Errorf("expected message %q, got %q", want, first.Message)
	}

	if first.Client != MAC("aa:bb:cc:dd:ee:ff") {
		t.Errorf("expected client MAC aa:bb:cc:dd:ee:ff, got %q", first.Client)
	}

	if first.Hostname != "laptop.local" {
		t.Errorf("expected hostname laptop.local, got %q", first.Hostname)
	}

	if want := time.UnixMilli(1751700000000); !first.DateTime.Equal(want) {
		t.Errorf("expected datetime %v, got %v", want, first.DateTime)
	}
}

// TestSession_GetAllEvents_PaginatesV2SystemLog verifies that fetching all
// events walks every page reported by total_page_count.
func TestSession_GetAllEvents_PaginatesV2SystemLog(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/system-log/") {
			notFoundJSON(w)
			return
		}

		var req SystemLogRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decoding system-log request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{
			"data": [{"id": "evt-page-%d", "key": "CLIENT_CONNECTED", "message_raw": "hello", "timestamp": %d}],
			"page_number": %d,
			"total_element_count": 2,
			"total_page_count": 2
		}`, req.PageNumber, 1751700000000+int64(req.PageNumber)*1000, req.PageNumber)
	})

	session, _ := newTestSession(t, handler)

	events, err := session.GetAllEvents()
	if err != nil {
		t.Fatalf("GetAllEvents failed: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events across 2 pages, got %d", len(events))
	}

	if events[0].ID != "evt-page-0" || events[1].ID != "evt-page-1" {
		t.Errorf("expected [evt-page-0 evt-page-1], got [%s %s]", events[0].ID, events[1].ID)
	}
}

// TestSession_DeadEndpointDoesNotTripCircuitBreaker verifies that an
// endpoint returning permanent HTTP errors (e.g. removed by a controller
// upgrade, like /stat/event was) never opens the session-wide circuit
// breaker, which would block healthy endpoints for every consumer.
func TestSession_DeadEndpointDoesNotTripCircuitBreaker(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stat/device") {
			okDataJSON(w, `[]`)
			return
		}
		notFoundJSON(w)
	})

	session, _ := newTestSession(t, handler)

	// Exceed the default MaxFailures (5) with back-to-back permanent errors.
	for i := 0; i < 8; i++ {
		if _, err := session.GetRecentEvents(); err == nil {
			t.Fatal("expected events fetch to fail, got nil error")
		}
	}

	if _, err := session.GetDevices(); err != nil {
		t.Fatalf("device fetch blocked after dead-endpoint failures: %v", err)
	}

	if got := session.CircuitBreakerStats().State; got != CircuitBreakerClosed {
		t.Errorf("expected circuit breaker to stay closed, got %v", got)
	}
}

// TestSession_GetUserByMAC_UsesStatUser verifies the by-MAC lookup uses the
// /stat/user/{mac} endpoint. The previous /rest/user/?mac= form is silently
// ignored by current UniFi Network releases, which return ALL users -- making
// Data[0] (and thus SetUserDetails) target an arbitrary wrong user.
func TestSession_GetUserByMAC_UsesStatUser(t *testing.T) {
	const mac = "f0:a2:25:b4:15:e8"

	var gotPath string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		okDataJSON(w, fmt.Sprintf(`[{"_id":"u1","mac":%q,"name":"printer"}]`, mac))
	})

	session, _ := newTestSession(t, handler)

	if _, err := session.GetUserByMAC(mac); err != nil {
		t.Fatalf("GetUserByMAC failed: %v", err)
	}

	if want := "/proxy/network/api/s/default/stat/user/" + mac; gotPath != want {
		t.Errorf("expected path %q, got %q", want, gotPath)
	}
}

// TestSession_GetUserByMac_RejectsWrongUser verifies the lookup fails loudly
// if the controller returns a different client than the requested MAC, rather
// than letting SetUserDetails modify an arbitrary user.
func TestSession_GetUserByMac_RejectsWrongUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		okDataJSON(w, `[{"_id":"u-other","mac":"00:08:22:e8:ad:fb","name":"someone-else"}]`)
	})

	session, _ := newTestSession(t, handler)

	if _, err := session.getUserByMac("f0:a2:25:b4:15:e8"); err == nil {
		t.Fatal("expected error when controller returns a different user, got nil")
	}
}

// TestSession_FailingEventsFetchDoesNotPoisonSession is the regression test
// for the sticky session error: a failing events fetch must not block
// subsequent device/client/user fetches.
func TestSession_FailingEventsFetchDoesNotPoisonSession(t *testing.T) {
	var deviceRequests, clientRequests, userRequests atomic.Int64

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/system-log/"),
			strings.Contains(r.URL.Path, "/stat/event"),
			strings.Contains(r.URL.Path, "/rest/event"):
			notFoundJSON(w)
		case strings.HasSuffix(r.URL.Path, "/stat/device"):
			deviceRequests.Add(1)
			okDataJSON(w, `[{"_id":"d1","mac":"11:22:33:44:55:66","name":"switch"}]`)
		case strings.HasSuffix(r.URL.Path, "/stat/sta"):
			clientRequests.Add(1)
			okDataJSON(w, `[]`)
		case strings.HasSuffix(r.URL.Path, "/rest/user"):
			userRequests.Add(1)
			okDataJSON(w, `[]`)
		default:
			notFoundJSON(w)
		}
	})

	session, _ := newTestSession(t, handler)

	if _, err := session.GetRecentEvents(); err == nil {
		t.Fatal("expected events fetch to fail (endpoint 404s), got nil error")
	}

	devices, err := session.GetDevices()
	if err != nil {
		t.Fatalf("device fetch after failed events fetch: %v", err)
	}

	if len(devices) != 1 || devices[0].Name != "switch" {
		t.Errorf("expected 1 device named switch, got %+v", devices)
	}

	if deviceRequests.Load() == 0 {
		t.Error("device fetch never reached the server (short-circuited by sticky session error)")
	}

	if _, err := session.GetClients(); err != nil {
		t.Fatalf("client fetch after failed events fetch: %v", err)
	}

	if clientRequests.Load() == 0 {
		t.Error("client fetch never reached the server (short-circuited by sticky session error)")
	}

	if _, err := session.GetAllClients(); err != nil {
		t.Fatalf("user fetch after failed events fetch: %v", err)
	}

	if userRequests.Load() == 0 {
		t.Error("user fetch never reached the server (short-circuited by sticky session error)")
	}

	// A second events failure still must not poison subsequent fetches.
	if _, err := session.GetRecentEvents(); err == nil {
		t.Fatal("expected events fetch to keep failing, got nil error")
	}

	if _, err := session.GetDevices(); err != nil {
		t.Fatalf("device fetch after second failed events fetch: %v", err)
	}
}

// TestSession_SetUserDetails_PreservesNetworkAndGroup verifies the update
// payload carries the user's existing network_id and usergroup_id instead of
// a hardcoded network and a blank group.
func TestSession_SetUserDetails_PreservesNetworkAndGroup(t *testing.T) {
	const mac = "aa:bb:cc:dd:ee:ff"

	var putBody string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/stat/user/"+mac):
			okDataJSON(w, fmt.Sprintf(
				`[{"_id":"u1","mac":%q,"name":"old-name","network_id":"net-123","usergroup_id":"grp-456"}]`, mac,
			))
		case r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/rest/user/u1"):
			b, _ := io.ReadAll(r.Body)
			putBody = string(b)
			okDataJSON(w, `[]`)
		default:
			notFoundJSON(w)
		}
	})

	session, _ := newTestSession(t, handler)

	if _, err := session.SetUserDetails(mac, "new-name", "10.0.0.9"); err != nil {
		t.Fatalf("SetUserDetails failed: %v", err)
	}

	if putBody == "" {
		t.Fatal("no PUT request reached the server")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(putBody), &payload); err != nil {
		t.Fatalf("PUT body is not valid JSON: %v\nbody: %s", err, putBody)
	}

	if got := payload["network_id"]; got != "net-123" {
		t.Errorf("expected user's own network_id net-123, got %v", got)
	}

	if got := payload["usergroup_id"]; got != "grp-456" {
		t.Errorf("expected user's own usergroup_id grp-456, got %v", got)
	}

	if got := payload["name"]; got != "new-name" {
		t.Errorf("expected name new-name, got %v", got)
	}

	if got := payload["fixed_ip"]; got != "10.0.0.9" {
		t.Errorf("expected fixed_ip 10.0.0.9, got %v", got)
	}
}
