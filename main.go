package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"keenetic-manager/keenetic"

	"github.com/gin-gonic/gin"
)

type Config struct {
	RouterIP   string
	RouterUser string
	RouterPass string
	WebUser    string
	WebPass    string
	WebTTL     string
}

var (
	cfg             Config
	routerClient    *keenetic.Client
	domainStorePath string
	domainStoreMu   sync.Mutex
)

// ---------- DEVICE PAGE ----------

type RciPolicyResponse map[string]struct {
	Description string `json:"description"`
}

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

type ScHostItem struct {
	Mac    string
	Access string
	Policy string
	Permit bool
	Deny   bool
}

type ScDhcpHost struct {
	Mac string `json:"mac"`
	IP  string `json:"ip"`
}

type IpHotspotHost struct {
	Mac  string
	IP   string
	Name string
	Link string
}

// show/sc/ip/hotspot
func parseScHotspotHosts(raw []byte) ([]ScHostItem, error) {
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

func parseScHostArray(v interface{}) ([]ScHostItem, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("host field is not an array")
	}

	var res []ScHostItem
	for _, el := range arr {
		obj, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		var item ScHostItem

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

// show/ip/hotspot
func parseIpHotspotHosts(raw []byte) ([]IpHotspotHost, error) {
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

func parseIpHostArray(v interface{}) ([]IpHotspotHost, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("host field is not an array")
	}

	var res []IpHotspotHost
	for _, el := range arr {
		obj, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		var item IpHotspotHost

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

// show/sc/ip/dhcp/host
func parseScDhcpHosts(raw []byte) ([]ScDhcpHost, error) {
	var arr []ScDhcpHost
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

func parseScDhcpArray(v interface{}) ([]ScDhcpHost, error) {
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("dhcp host field is not an array")
	}

	var res []ScDhcpHost
	for _, el := range arr {
		obj, ok := el.(map[string]interface{})
		if !ok {
			continue
		}
		var item ScDhcpHost

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

func initRouter() {
	var err error
	routerClient, err = keenetic.NewClient(cfg.RouterIP, cfg.RouterUser, cfg.RouterPass)
	if err != nil {
		log.Fatalf("Client init error: %v", err)
	}
	if err := routerClient.Login(); err != nil {
		log.Printf("Warning: Initial login failed: %v", err)
	}
}

func fetchRouterData() ([]Device, []Policy, error) {
	// Batch commands:
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
	if err := routerClient.SendBatch(batchPayload, &batchResp); err != nil {
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
	var policyData RciPolicyResponse
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

func apiGetDevices(c *gin.Context) {
	devices, policies, err := fetchRouterData()
	if err != nil {
		log.Printf("Fetch error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"devices": devices, "policies": policies})
}

func apiSetPolicy(c *gin.Context) {
	var req struct {
		MAC      string `json:"mac"`
		PolicyID string `json:"policy_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	if req.PolicyID == "_DENY_" {
		if err := routerClient.RciPost("ip/hotspot/host", map[string]string{
			"mac":    req.MAC,
			"access": "deny",
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if req.PolicyID == "" || req.PolicyID == "default" {
		payload := map[string]interface{}{
			"mac":    req.MAC,
			"permit": true,
			"policy": map[string]bool{"no": true},
		}
		if err := routerClient.RciPost("ip/hotspot/host", payload); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := routerClient.RciPost("ip/hotspot/host", map[string]string{
			"mac":    req.MAC,
			"access": "permit",
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := routerClient.RciPost("ip/hotspot/host", map[string]interface{}{
			"mac":    req.MAC,
			"policy": req.PolicyID,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	_ = routerClient.RciPost("system/configuration/save", nil)
	c.JSON(200, gin.H{"status": "ok"})
}

func apiSetStaticIP(c *gin.Context) {
	var req struct {
		MAC string `json:"mac"`
		IP  string `json:"ip"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	if req.IP == "" {
		if err := routerClient.RciPost("ip/dhcp/host", map[string]interface{}{
			"mac": req.MAC,
			"no":  true,
		}); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := routerClient.RciPost("ip/dhcp/host", map[string]string{
			"mac": req.MAC,
			"ip":  req.IP,
		}); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	_ = routerClient.RciPost("system/configuration/save", nil)
	c.JSON(200, gin.H{"status": "ok"})
}

// ---------- SECOND PAGE: DOMAIN ROUTES ----------

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
}

type DomainStore struct {
    Interface   string             `json:"interface"`
    Entries     []DomainRouteEntry `json:"entries"`
    AutoRefresh bool               `json:"auto_refresh"`
}

func loadDomainStore() (*DomainStore, error) {
	domainStoreMu.Lock()
	defer domainStoreMu.Unlock()

	data, err := os.ReadFile(domainStorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &DomainStore{Entries: []DomainRouteEntry{}}, nil
		}
		return nil, err
	}

	var store DomainStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Entries == nil {
		store.Entries = []DomainRouteEntry{}
	}
	return &store, nil
}

func saveDomainStore(store *DomainStore) error {
	domainStoreMu.Lock()
	defer domainStoreMu.Unlock()

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(domainStorePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(domainStorePath, data, 0o644)
}

func findDomainEntry(store *DomainStore, domain string) (*DomainRouteEntry, int) {
	for i := range store.Entries {
		if strings.EqualFold(store.Entries[i].Domain, domain) {
			return &store.Entries[i], i
		}
	}
	return nil, -1
}

func lookupDomainIPv4(domain string) ([]string, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	var res []string
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			s := v4.String()
			if s != "" && !seen[s] {
				seen[s] = true
				res = append(res, s)
			}
		}
	}
	return res, nil
}

func applyRoutesForDomain(store *DomainStore, entry *DomainRouteEntry) error {
	newIf := strings.TrimSpace(store.Interface)
	if newIf == "" {
		return fmt.Errorf("no interface selected")
	}

	oldIf := entry.AppliedInterface
	if oldIf == "" {
		oldIf = newIf
	}

	oldSet := make(map[string]bool)
	for _, ip := range entry.AppliedIPs {
		oldSet[ip] = true
	}
	newSet := make(map[string]bool)
	for _, ip := range entry.IPs {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		newSet[ip] = true
	}

	for ip := range oldSet {
		if !newSet[ip] && ip != "" && oldIf != "" {
			payload := map[string]interface{}{
				"host":      ip,
				"interface": oldIf,
				"no":        true,
			}
			if err := routerClient.RciPost("ip/route", payload); err != nil {
				return err
			}
		}
	}

	comment := fmt.Sprintf("%s - auto generated", entry.Domain)
	for ip := range newSet {
		if oldSet[ip] {
			continue
		}
		payload := map[string]interface{}{
			"gateway":   "",
			"auto":      true,
			"reject":    false,
			"comment":   comment,
			"interface": newIf,
			"host":      ip,
		}
		if err := routerClient.RciPost("ip/route", payload); err != nil {
			return err
		}
	}

	entry.AppliedInterface = newIf
	entry.AppliedIPs = make([]string, 0, len(newSet))
	for ip := range newSet {
		entry.AppliedIPs = append(entry.AppliedIPs, ip)
	}
	sort.Strings(entry.AppliedIPs)

	return routerClient.RciPost("system/configuration/save", nil)
}

func apiRoutesGetData(c *gin.Context) {
	var ifResp ShowInterfaceResponse
	if err := routerClient.RciGet("show/interface", &ifResp); err != nil {
		log.Printf("Interfaces fetch error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var interfaces []RouterInterface
	for id, inf := range ifResp {
		include := false
		for _, t := range inf.Traits {
			if t == "Ip" {
				include = true
				break
			}
		}
		if !include {
			continue
		}
		name := inf.InterfaceName
		if name == "" {
			name = id
		}
		interfaces = append(interfaces, RouterInterface{
			ID:          id,
			Name:        name,
			Description: inf.Description,
		})
	}
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Name < interfaces[j].Name
	})

	store, err := loadDomainStore()
	if err != nil {
		log.Printf("Domain store load error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

    type Resp struct {
        Interfaces        []RouterInterface  `json:"interfaces"`
        SelectedInterface string             `json:"selected_interface"`
        Domains           []DomainRouteEntry `json:"domains"`
        AutoRefresh       bool               `json:"auto_refresh"`
    }
    c.JSON(200, Resp{
        Interfaces:        interfaces,
        SelectedInterface: store.Interface,
        Domains:           store.Entries,
        AutoRefresh:       store.AutoRefresh,
    })
}

func apiRoutesSetInterface(c *gin.Context) {
	var req struct {
		Interface string `json:"interface"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Interface) == "" {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}

	store, err := loadDomainStore()
	if err != nil {
		log.Printf("Domain store load error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	store.Interface = strings.TrimSpace(req.Interface)
	if err := saveDomainStore(store); err != nil {
		log.Printf("Domain store save error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "ok"})
}

func apiRoutesDomainAdd(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := loadDomainStore()
	if err != nil {
		log.Printf("Domain store load error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(store.Interface) == "" {
		c.JSON(400, gin.H{"error": "interface not selected"})
		return
	}

	ips, err := lookupDomainIPv4(domain)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("lookup failed: %v", err)})
		return
	}

	entry, idx := findDomainEntry(store, domain)
	nowStr := time.Now().UTC().Format(time.RFC3339)

	if entry == nil {
		store.Entries = append(store.Entries, DomainRouteEntry{
			Domain:     domain,
			IPs:        ips,
			LastLookup: nowStr,
		})
		entry = &store.Entries[len(store.Entries)-1]
	} else {
		entry.Domain = domain
		entry.IPs = ips
		entry.LastLookup = nowStr
		store.Entries[idx] = *entry
	}

	if err := applyRoutesForDomain(store, entry); err != nil {
		log.Printf("Apply routes error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	if err := saveDomainStore(store); err != nil {
		log.Printf("Domain store save error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"entry": entry})
}

func apiRoutesDomainLookup(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := loadDomainStore()
	if err != nil {
		log.Printf("Domain store load error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	entry, idx := findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(404, gin.H{"error": "domain not found"})
		return
	}

	ips, err := lookupDomainIPv4(domain)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("lookup failed: %v", err)})
		return
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	entry.IPs = ips
	entry.LastLookup = nowStr
	store.Entries[idx] = *entry

	if err := saveDomainStore(store); err != nil {
		log.Printf("Domain store save error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"entry": entry})
}

func apiRoutesDomainEdit(c *gin.Context) {
	var req struct {
		Domain string   `json:"domain"`
		IPs    []string `json:"ips"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := loadDomainStore()
	if err != nil {
		log.Printf("Domain store load error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	entry, idx := findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(404, gin.H{"error": "domain not found"})
		return
	}

	var ips []string
	for _, ip := range req.IPs {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	entry.IPs = ips
	store.Entries[idx] = *entry

	if err := saveDomainStore(store); err != nil {
		log.Printf("Domain store save error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"entry": entry})
}

func apiRoutesDomainApply(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := loadDomainStore()
	if err != nil {
		log.Printf("Domain store load error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(store.Interface) == "" {
		c.JSON(400, gin.H{"error": "interface not selected"})
		return
	}

	entry, idx := findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(404, gin.H{"error": "domain not found"})
		return
	}

	if err := applyRoutesForDomain(store, entry); err != nil {
		log.Printf("Apply routes error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	store.Entries[idx] = *entry
	if err := saveDomainStore(store); err != nil {
		log.Printf("Domain store save error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"entry": entry})
}

func apiRoutesDomainDelete(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(400, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := loadDomainStore()
	if err != nil {
		log.Printf("Domain store load error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	entry, idx := findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(404, gin.H{"error": "domain not found"})
		return
	}

	oldIf := entry.AppliedInterface
	if oldIf == "" {
		oldIf = store.Interface
	}
	for _, ip := range entry.AppliedIPs {
		if ip == "" || oldIf == "" {
			continue
		}
		payload := map[string]interface{}{
			"host":      ip,
			"interface": oldIf,
			"no":        true,
		}
		if err := routerClient.RciPost("ip/route", payload); err != nil {
			log.Printf("Delete route error: %v", err)
		}
	}
	_ = routerClient.RciPost("system/configuration/save", nil)

	store.Entries = append(store.Entries[:idx], store.Entries[idx+1:]...)
	if err := saveDomainStore(store); err != nil {
		log.Printf("Domain store save error: %v", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"status": "ok"})
}

// ---------- MAIN ----------

func main() {
	cfg.RouterIP = os.Getenv("ROUTER_IP")
	cfg.RouterUser = os.Getenv("ROUTER_USER")
	cfg.RouterPass = os.Getenv("ROUTER_PASS")
	cfg.WebUser = os.Getenv("WEB_USER")
	cfg.WebPass = os.Getenv("WEB_PASS")
	cfg.WebTTL = os.Getenv("WEB_SESSION_TTL")

	if cfg.RouterIP == "" {
		cfg.RouterIP = "192.168.1.1"
	}
	if cfg.WebUser == "" {
		log.Fatal("WEB_USER required")
	}
	if cfg.WebTTL == "" {
		cfg.WebTTL = "24h"
	}
	if _, err := time.ParseDuration(cfg.WebTTL); err != nil {
		cfg.WebTTL = "24h"
	}

	domainStorePath = os.Getenv("DOMAIN_STORE_PATH")
	if domainStorePath == "" {
		domainStorePath = "/data/domains.json"
	}

	initRouter()

	r := gin.Default()
	r.Delims("[[", "]]")
	r.Static("/assets", "./static/assets")
	r.LoadHTMLFiles("./static/index.html")

	auth := gin.BasicAuth(gin.Accounts{cfg.WebUser: cfg.WebPass})

	r.GET("/", auth, func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"User": cfg.WebUser,
			"Host": cfg.RouterIP,
			"TTL":  cfg.WebTTL,
		})
	})

	api := r.Group("/api", auth)
	{
		api.GET("/data", apiGetDevices)
		api.POST("/policy", apiSetPolicy)
		api.POST("/static_ip", apiSetStaticIP)

		routes := api.Group("/routes")
		{
			routes.GET("/data", apiRoutesGetData)
			routes.POST("/interface", apiRoutesSetInterface)
			routes.POST("/domain/add", apiRoutesDomainAdd)
			routes.POST("/domain/lookup", apiRoutesDomainLookup)
			routes.POST("/domain/edit", apiRoutesDomainEdit)
			routes.POST("/domain/apply", apiRoutesDomainApply)
			routes.POST("/domain/delete", apiRoutesDomainDelete)
			routes.POST("/sync_all", apiRoutesSyncAll)
            routes.POST("/auto_refresh", apiRoutesSetAutoRefresh)
		}
	}

    scheduleAutoRefresh()

	fmt.Println("Starting Keenetic Go (Hybrid API) on port 800...")
	_ = r.Run(":800")
}

func apiRoutesSetAutoRefresh(c *gin.Context) {
    var req struct {
        Enabled bool `json:"enabled"`
    }
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Bad request"})
        return
    }

    store, err := loadDomainStore()
    if err != nil {
        log.Printf("Domain store load error: %v", err)
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    store.AutoRefresh = req.Enabled
    if err := saveDomainStore(store); err != nil {
        log.Printf("Domain store save error: %v", err)
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"status": "ok"})
}

func sameIPSet(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    set := make(map[string]bool, len(a))
    for _, ip := range a {
        set[ip] = true
    }
    for _, ip := range b {
        if !set[ip] {
            return false
        }
    }
    return true
}

func syncAllDomains(force bool) (*DomainStore, error) {
    store, err := loadDomainStore()
    if err != nil {
        return nil, err
    }

    if !force && !store.AutoRefresh {
        return store, nil
    }
    if strings.TrimSpace(store.Interface) == "" {
        return nil, fmt.Errorf("interface not selected")
    }

    for i := range store.Entries {
        e := &store.Entries[i]
        if strings.TrimSpace(e.Domain) == "" {
            continue
        }

        ips, err := lookupDomainIPv4(e.Domain)
        if err != nil {
            log.Printf("syncAll lookup error for %s: %v", e.Domain, err)
            continue
        }
        nowStr := time.Now().UTC().Format(time.RFC3339)

        changed := !sameIPSet(e.IPs, ips)
        e.IPs = ips
        e.LastLookup = nowStr

        if changed {
            if err := applyRoutesForDomain(store, e); err != nil {
                log.Printf("syncAll apply error for %s: %v", e.Domain, err)
            }
        }
    }

    if err := saveDomainStore(store); err != nil {
        return nil, err
    }
    return store, nil
}

func apiRoutesSyncAll(c *gin.Context) {
    store, err := syncAllDomains(true)
    if err != nil {
        log.Printf("SyncAll error: %v", err)
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"entries": store.Entries})
}

func scheduleAutoRefresh() {
    go func() {
        for {
            now := time.Now().UTC()
            next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
            time.Sleep(next.Sub(now))

            _, err := syncAllDomains(false)
            if err != nil {
                log.Printf("Auto syncAll error: %v", err)
            }
        }
    }()
}