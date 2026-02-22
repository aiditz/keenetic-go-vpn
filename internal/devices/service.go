// internal/devices/service.go
package devices

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"keenetic-go-vpn/keenetic"
)

type Service struct {
	client *keenetic.Client
}

func NewService(c *keenetic.Client) *Service {
	return &Service{client: c}
}

func (s *Service) FetchRouterData() ([]Device, []Policy, error) {
	// Batch:
	// 0: show/sc/ip/hotspot
	// 1: show/ip/hotspot
	// 2: show/sc/ip/dhcp/host
	// 3: show/ip/policy
	batchPayload := []interface{}{
		map[string]interface{}{"show": map[string]interface{}{"sc": map[string]interface{}{"ip": map[string]interface{}{"hotspot": map[string]interface{}{}}}}},
		map[string]interface{}{"show": map[string]interface{}{"ip": map[string]interface{}{"hotspot": map[string]interface{}{}}}},
		map[string]interface{}{"show": map[string]interface{}{"sc": map[string]interface{}{"ip": map[string]interface{}{"dhcp": map[string]interface{}{"host": map[string]interface{}{}}}}}},
		map[string]interface{}{"show": map[string]interface{}{"ip": map[string]interface{}{"policy": map[string]interface{}{}}}},
	}

	var batchResp []json.RawMessage
	if err := s.client.SendBatch(batchPayload, &batchResp); err != nil {
		return nil, nil, fmt.Errorf("batch fetch failed: %v", err)
	}
	if len(batchResp) != 4 {
		return nil, nil, fmt.Errorf("unexpected batch response length: %d", len(batchResp))
	}

	// 0: sc hotspot
	hostRules := make(map[string]map[string]string)
	if len(batchResp[0]) > 0 {
		scHosts, err := parseScHotspotHosts(batchResp[0])
		if err != nil {
			log.Printf("SC hotspot parse warning: %v", err)
		} else {
			for _, h := range scHosts {
				mac := strings.ToLower(h.Mac)
				if mac == "" {
					continue
				}
				if _, exists := hostRules[mac]; !exists {
					hostRules[mac] = map[string]string{
						"access": "permit",
						"policy": "",
					}
				}

				acc := strings.ToLower(h.Access)
				if acc == "" {
					if h.Deny {
						acc = "deny"
					} else if h.Permit {
						acc = "permit"
					}
				}
				if acc != "" {
					hostRules[mac]["access"] = acc
				}

				if h.Policy != "" {
					hostRules[mac]["policy"] = h.Policy
				}
			}
		}
	}

	// 1: ip hotspot
	ipHosts, err := parseIpHotspotHosts(batchResp[1])
	if err != nil {
		return nil, nil, fmt.Errorf("ip hotspot parse failed: %v", err)
	}

	// 2: sc dhcp host
	staticIps := make(map[string]string)
	if len(batchResp[2]) > 0 {
		dhcpHosts, err := parseScDhcpHosts(batchResp[2])
		if err != nil {
			log.Printf("SC DHCP parse warning: %v", err)
		} else {
			for _, h := range dhcpHosts {
				if h.Mac == "" || h.IP == "" {
					continue
				}
				staticIps[strings.ToLower(h.Mac)] = h.IP
			}
		}
	}

	// 3: ip policy
	var policyData rciPolicyResponse
	if len(batchResp[3]) > 0 {
		var root interface{}
		if err := json.Unmarshal(batchResp[3], &root); err != nil {
			log.Printf("Policy JSON unmarshal warning: %v", err)
		} else if m, ok := root.(map[string]interface{}); ok {
			var sub interface{} = m
			if showV, ok := m["show"].(map[string]interface{}); ok {
				if ipV, ok := showV["ip"].(map[string]interface{}); ok {
					if polV, ok := ipV["policy"]; ok {
						sub = polV
					}
				}
			}
			if subBytes, err := json.Marshal(sub); err == nil {
				_ = json.Unmarshal(subBytes, &policyData)
			}
		}
	}

	policyMap := make(map[string]string)
	var policies []Policy
	for id, details := range policyData {
		desc := details.Description
		if desc == "" {
			desc = id
		}
		policyMap[id] = desc
		policies = append(policies, Policy{ID: id, Desc: desc})
	}

	devicesMap := make(map[string]*Device)

	// init from runtime hotspot
	for _, h := range ipHosts {
		mac := strings.ToLower(h.Mac)
		if mac == "" {
			continue
		}

		d := &Device{
			MAC:    mac,
			Name:   "Unknown",
			Access: "PERMIT",
			IP:     "-",
			Online: false,
		}

		if h.Name != "" && h.Name != mac {
			d.Name = h.Name
		}
		if h.IP != "" {
			d.IP = h.IP
		}
		if strings.ToLower(h.Link) == "up" {
			d.Online = true
		}
		if sip, ok := staticIps[mac]; ok {
			d.StaticIP = sip
		}

		devicesMap[mac] = d
	}

	// add devices that only exist in static DHCP
	for mac, sip := range staticIps {
		if _, ok := devicesMap[mac]; !ok {
			devicesMap[mac] = &Device{
				MAC:      mac,
				Name:     "Unknown",
				Access:   "PERMIT",
				IP:       "-",
				Online:   false,
				StaticIP: sip,
			}
		}
	}

	// apply hotspot rules
	for mac, rule := range hostRules {
		lm := strings.ToLower(mac)
		d, ok := devicesMap[lm]
		if !ok {
			d = &Device{
				MAC:    lm,
				Name:   "Unknown",
				Access: "PERMIT",
				IP:     "-",
				Online: false,
			}
			devicesMap[lm] = d
		}

		if acc, ok := rule["access"]; ok && acc != "" {
			d.Access = strings.ToUpper(acc)
		}
		if pol, ok := rule["policy"]; ok && pol != "" {
			d.PolicyID = pol
		}
	}

	var devices []Device
	for _, d := range devicesMap {
		d.PolicyDesc = "Default"
		if d.PolicyID != "" {
			if val, ok := policyMap[d.PolicyID]; ok {
				d.PolicyDesc = val
			} else {
				d.PolicyDesc = d.PolicyID
			}
		}
		devices = append(devices, *d)
	}

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].Online != devices[j].Online {
			return devices[i].Online
		}
		rankI := getPolicyRank(devices[i])
		rankJ := getPolicyRank(devices[j])
		if rankI != rankJ {
			return rankI < rankJ
		}
		return strings.ToLower(devices[i].Name) < strings.ToLower(devices[j].Name)
	})

	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Desc < policies[j].Desc
	})

	return devices, policies, nil
}

func getPolicyRank(d Device) int {
	if d.Access == "DENY" {
		return 2
	}
	if d.PolicyID != "" {
		return 1
	}
	return 0
}

func parseScHotspotHosts(raw []byte) ([]scHostItem, error) {
	var root interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected sc hotspot JSON root type")
	}

	if hv, ok := m["host"]; ok {
		return parseScHostArray(hv)
	}

	if showV, ok := m["show"].(map[string]interface{}); ok {
		if scV, ok := showV["sc"].(map[string]interface{}); ok {
			if ipV, ok := scV["ip"].(map[string]interface{}); ok {
				if hsV, ok := ipV["hotspot"].(map[string]interface{}); ok {
					if hv, ok := hsV["host"]; ok {
						return parseScHostArray(hv)
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no host[] found in show/sc/ip/hotspot response")
}

func parseScHostArray(v interface{}) ([]scHostItem, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("host field is not an array")
	}

	var res []scHostItem
	for _, el := range arr {
		obj, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		var item scHostItem

		if mac, ok := obj["mac"].(string); ok {
			item.Mac = mac
		}
		if acc, ok := obj["access"].(string); ok {
			item.Access = acc
		}
		if pol, ok := obj["policy"].(string); ok {
			item.Policy = pol
		}
		if p, ok := obj["permit"].(bool); ok {
			item.Permit = p
		}
		if d, ok := obj["deny"].(bool); ok {
			item.Deny = d
		}

		if item.Mac != "" {
			res = append(res, item)
		}
	}
	return res, nil
}

func parseIpHotspotHosts(raw []byte) ([]ipHotspotHost, error) {
	var root interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected ip hotspot JSON root type")
	}

	if hv, ok := m["host"]; ok {
		return parseIpHostArray(hv)
	}

	if showV, ok := m["show"].(map[string]interface{}); ok {
		if ipV, ok := showV["ip"].(map[string]interface{}); ok {
			if hsV, ok := ipV["hotspot"].(map[string]interface{}); ok {
				if hv, ok := hsV["host"]; ok {
					return parseIpHostArray(hv)
				}
			}
		}
	}

	return nil, fmt.Errorf("no host[] found in show/ip/hotspot response")
}

func parseIpHostArray(v interface{}) ([]ipHotspotHost, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("host field is not an array")
	}

	var res []ipHotspotHost
	for _, el := range arr {
		obj, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		var item ipHotspotHost

		if mac, ok := obj["mac"].(string); ok {
			item.Mac = mac
		}
		if ip, ok := obj["ip"].(string); ok {
			item.IP = ip
		}
		if name, ok := obj["name"].(string); ok {
			item.Name = name
		}
		if link, ok := obj["link"].(string); ok {
			item.Link = link
		}

		if item.Mac != "" {
			res = append(res, item)
		}
	}
	return res, nil
}

func parseScDhcpHosts(raw []byte) ([]scDhcpHost, error) {
	var arr []scDhcpHost
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	var root interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	m, ok := root.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected sc dhcp JSON root type")
	}

	if hv, ok := m["host"]; ok {
		return parseScDhcpArray(hv)
	}

	if showV, ok := m["show"].(map[string]interface{}); ok {
		if scV, ok := showV["sc"].(map[string]interface{}); ok {
			if ipV, ok := scV["ip"].(map[string]interface{}); ok {
				if dhcpV, ok := ipV["dhcp"].(map[string]interface{}); ok {
					if hv, ok := dhcpV["host"]; ok {
						return parseScDhcpArray(hv)
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no host[] found in show/sc/ip/dhcp/host response")
}

func parseScDhcpArray(v interface{}) ([]scDhcpHost, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("dhcp host field is not an array")
	}

	var res []scDhcpHost
	for _, el := range arr {
		obj, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		var item scDhcpHost

		if mac, ok := obj["mac"].(string); ok {
			item.Mac = mac
		}
		if ip, ok := obj["ip"].(string); ok {
			item.IP = ip
		}
		if item.Mac != "" && item.IP != "" {
			res = append(res, item)
		}
	}
	return res, nil
}