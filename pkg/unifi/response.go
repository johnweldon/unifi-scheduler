package unifi

// ClientResponse encapsulates a UniFi http response.
type ClientResponse struct {
	Meta Meta     `json:"meta,omitempty"`
	Data []Client `json:"data,omitempty"`
}

// DeviceResponse encapsulates a UniFi http response.
type DeviceResponse struct {
	Meta Meta     `json:"meta,omitempty"`
	Data []Device `json:"data,omitempty"`
}

// EventResponse encapsulates a UniFi http response.
type EventResponse struct {
	Meta Meta    `json:"meta,omitempty"`
	Data []Event `json:"data,omitempty"`
}

// Meta encapsulates basic meta from response.
type Meta struct {
	RC      string `json:"rc,omitempty"`
	Count   int64  `json:"count,omitempty"`
	Message string `json:"msg,omitempty"`
}
