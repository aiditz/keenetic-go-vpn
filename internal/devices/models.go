package devices

type Device struct {
	MAC        string `json:"mac"`
	IP         string `json:"ip"`
	StaticIP   string `json:"static_ip"`
	Name       string `json:"name"`
	Online     bool   `json:"online"`
	Access     string `json:"access"`
	PolicyID   string `json:"policy_id"`
	PolicyDesc string `json:"policy_desc"`
}

type Policy struct {
	ID   string `json:"id"`
	Desc string `json:"desc"`
}

// JSON from /rci/show/ip/policy -> { "Policy0": { "description": "NoVPN", ... }, ... }
type rciPolicyResponse map[string]struct {
	Description string `json:"description"`
}

// internal structs for show/sc/ip/hotspot, show/ip/hotspot, show/sc/ip/dhcp/host
type scHostItem struct {
	Mac    string
	Access string
	Policy string
	Permit bool
	Deny   bool
}

type scDhcpHost struct {
	Mac string `json:"mac"`
	IP  string `json:"ip"`
}

type ipHotspotHost struct {
	Mac  string
	IP   string
	Name string
	Link string
}