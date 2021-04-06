package unifi

// Response encapsulates a UniFi http response.
type Response struct {
	Meta struct {
		RC string `json:"rc,omitempty"`
	}
	Data []Client `json:"data,omitempty"`
}
