package unifi

// ClientResponse encapsulates a UniFi http response.
type ClientResponse struct {
	Meta struct {
		RC string `json:"rc,omitempty"`
	}
	Data []Client `json:"data,omitempty"`
}

// DeviceResponse encapsulates a UniFi http response.
type DeviceResponse struct {
	Meta struct {
		RC string `json:"rc,omitempty"`
	}
	Data []Device `json:"data,omitempty"`
}
