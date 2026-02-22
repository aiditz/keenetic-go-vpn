package routes

type RouterInterface struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type InterfaceInfo struct {
	ID            string   `json:"id"`
	InterfaceName string   `json:"interface-name"`
	Description   string   `json:"description"`
	Traits        []string `json:"traits"`
}

type ShowInterfaceResponse map[string]InterfaceInfo

type DomainRouteEntry struct {
	Domain           string   `json:"domain"`
	IPs              []string `json:"ips"`
	LastLookup       string   `json:"last_lookup"`
	AppliedIPs       []string `json:"applied_ips,omitempty"`
	AppliedInterface string   `json:"applied_interface,omitempty"`
	Disabled         bool     `json:"disabled,omitempty"`
}

type DomainStore struct {
	Interface   string             `json:"interface"`
	Entries     []DomainRouteEntry `json:"entries"`
	AutoRefresh bool               `json:"auto_refresh"`
}