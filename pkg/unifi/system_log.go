package unifi

import (
	"regexp"
	"time"
)

// systemLogCategories are the category enum values accepted by the v2
// system-log API on current UniFi Network releases. Legacy values such as
// MONITORING, INTERNET, or SYSTEM are rejected with HTTP 400.
var systemLogCategories = []string{
	"SECURITY",
	"UNIFI_DEVICES",
	"SOFTWARE_UPDATES",
	"VPN",
	"POWER",
	"UNIFI_ETHERNET_PORTS",
	"CLIENT_DEVICES",
	"UNKNOWN",
	"AUDIT",
	"INTERNET_AND_WAN",
}

// SystemLogRequest is the request body for the v2 system-log API
// (POST /v2/api/site/{site}/system-log/all).
type SystemLogRequest struct {
	SearchText    string   `json:"searchText,omitempty"`
	Severities    []string `json:"severities,omitempty"`
	TimestampFrom int64    `json:"timestampFrom,omitempty"`
	TimestampTo   int64    `json:"timestampTo,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Subcategories []string `json:"subcategories,omitempty"`
	Events        []string `json:"events,omitempty"`
	PageNumber    int      `json:"pageNumber"`
	PageSize      int      `json:"pageSize"`
}

// SystemLogResponse is the paginated response from the v2 system-log API.
type SystemLogResponse struct {
	Data              []SystemLogEntry `json:"data,omitempty"`
	PageNumber        int              `json:"page_number,omitempty"`
	TotalElementCount int              `json:"total_element_count,omitempty"`
	TotalPageCount    int              `json:"total_page_count,omitempty"`
}

// SystemLogEntry is a single v2 system-log event.
type SystemLogEntry struct {
	ID          string                    `json:"id,omitempty"`
	Key         string                    `json:"key,omitempty"`
	Category    string                    `json:"category,omitempty"`
	Subcategory string                    `json:"subcategory,omitempty"`
	Event       string                    `json:"event,omitempty"`
	MessageRaw  string                    `json:"message_raw,omitempty"`
	TitleRaw    string                    `json:"title_raw,omitempty"`
	Parameters  map[string]SystemLogParam `json:"parameters,omitempty"`
	Severity    string                    `json:"severity,omitempty"`
	Status      string                    `json:"status,omitempty"`
	Target      string                    `json:"target,omitempty"`
	Timestamp   int64                     `json:"timestamp,omitempty"`
	Type        string                    `json:"type,omitempty"`
}

// SystemLogParam is contextual data attached to a system-log entry, keyed by
// placeholder name (e.g. CLIENT, DEVICE, WLAN).
type SystemLogParam struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	IP       string `json:"ip,omitempty"`
	Model    string `json:"model,omitempty"`
}

var systemLogPlaceholder = regexp.MustCompile(`\{(\w+)\}`)

// Message renders the raw message, substituting {PLACEHOLDER} tokens with the
// matching parameter's name, hostname, or id.
func (e SystemLogEntry) Message() string {
	raw := e.MessageRaw
	if raw == "" {
		raw = e.TitleRaw
	}

	return systemLogPlaceholder.ReplaceAllStringFunc(raw, func(match string) string {
		param, ok := e.Parameters[match[1:len(match)-1]]

		switch {
		case !ok:
			return match
		case param.Name != "":
			return param.Name
		case param.Hostname != "":
			return param.Hostname
		case param.ID != "":
			return param.ID
		}

		return match
	})
}

// ToEvent maps a v2 system-log entry onto the legacy Event shape used
// throughout the CLI and the NATS agent.
func (e SystemLogEntry) ToEvent() Event {
	evt := Event{
		ID:        e.ID,
		Key:       EventType(e.Key),
		Message:   e.Message(),
		Subsystem: e.Category,
		DateTime:  time.UnixMilli(e.Timestamp),
		TimeStamp: TimeStamp(e.Timestamp),
	}

	if client, ok := e.Parameters["CLIENT"]; ok {
		evt.Client = MAC(client.ID)
		evt.Hostname = client.Hostname
		evt.Name = client.Name
		evt.IP = IP(client.IP)
	}

	return evt
}
