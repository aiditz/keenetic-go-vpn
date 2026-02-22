// internal/routes/routes.go
package routes

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

func buildRouteComment(domains []string) string {
	if len(domains) == 0 {
		return "[GOVPN]"
	}
	uniq := make(map[string]bool, len(domains))
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		uniq[d] = true
	}
	list := make([]string, 0, len(uniq))
	for d := range uniq {
		list = append(list, d)
	}
	sort.Strings(list)
	return "[GOVPN] " + strings.Join(list, ", ")
}

func containsIP(list []string, ip string) bool {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return false
	}
	for _, v := range list {
		if strings.TrimSpace(v) == ip {
			return true
		}
	}
	return false
}

func removeIP(list []string, ip string) []string {
	ip = strings.TrimSpace(ip)
	if ip == "" || len(list) == 0 {
		return list
	}
	out := list[:0]
	for _, v := range list {
		if strings.TrimSpace(v) != ip {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Updates route on router for a single IP using aggregated domains from store.
func updateRoutesForIP(store *DomainStore, ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	iface := strings.TrimSpace(store.Interface)
	if iface == "" {
		return fmt.Errorf("interface not selected")
	}

	activeDomains := []string{}
	routeExists := false
	oldIf := ""

	for i := range store.Entries {
		e := &store.Entries[i]

		if containsIP(e.AppliedIPs, ip) {
			routeExists = true
			if e.AppliedInterface != "" {
				oldIf = e.AppliedInterface
			}
		}

		hasIp := false
		for _, ip2 := range e.IPs {
			if strings.TrimSpace(ip2) == ip {
				hasIp = true
				break
			}
		}
		if hasIp && !e.Disabled {
			activeDomains = append(activeDomains, e.Domain)
		}
	}

	if oldIf == "" {
		oldIf = iface
	}

	if len(activeDomains) == 0 {
		if routeExists {
			payloadNo := map[string]interface{}{
				"host":      ip,
				"interface": oldIf,
				"no":        true,
			}
			if err := svc.client.RciPost("ip/route", payloadNo); err != nil {
				return err
			}
		}
		for i := range store.Entries {
			e := &store.Entries[i]
			e.AppliedIPs = removeIP(e.AppliedIPs, ip)
			if len(e.AppliedIPs) == 0 {
				e.AppliedInterface = ""
			}
		}
		return nil
	}

	if routeExists {
		payloadNo := map[string]interface{}{
			"host":      ip,
			"interface": oldIf,
			"no":        true,
		}
		if err := svc.client.RciPost("ip/route", payloadNo); err != nil {
			return err
		}
	}

	comment := buildRouteComment(activeDomains)
	payloadAdd := map[string]interface{}{
		"gateway":   "",
		"auto":      true,
		"reject":    false,
		"comment":   comment,
		"interface": iface,
		"host":      ip,
	}
	if err := svc.client.RciPost("ip/route", payloadAdd); err != nil {
		return err
	}

	for i := range store.Entries {
		e := &store.Entries[i]

		hasIp := false
		for _, ip2 := range e.IPs {
			if strings.TrimSpace(ip2) == ip {
				hasIp = true
				break
			}
		}
		if !hasIp || e.Disabled {
			e.AppliedIPs = removeIP(e.AppliedIPs, ip)
			if len(e.AppliedIPs) == 0 {
				e.AppliedInterface = ""
			}
			continue
		}

		if !containsIP(e.AppliedIPs, ip) {
			e.AppliedIPs = append(e.AppliedIPs, ip)
		}
		e.AppliedInterface = iface
	}

	return nil
}

func applyRoutesForDomain(store *DomainStore, entry *DomainRouteEntry) error {
	entry.Disabled = false

	ipSet := make(map[string]bool)
	for _, ip := range entry.IPs {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			ipSet[ip] = true
		}
	}

	for ip := range ipSet {
		if err := updateRoutesForIP(store, ip); err != nil {
			return err
		}
	}

	if err := svc.saveDomainStore(store); err != nil {
		return err
	}
	return svc.client.RciPost("system/configuration/save", nil)
}

func removeRoutesForDomain(store *DomainStore, entry *DomainRouteEntry) error {
	ipSet := make(map[string]bool)
	for _, ip := range entry.IPs {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			ipSet[ip] = true
		}
	}

	for ip := range ipSet {
		if err := updateRoutesForIP(store, ip); err != nil {
			return err
		}
	}

	if err := svc.saveDomainStore(store); err != nil {
		return err
	}
	return svc.client.RciPost("system/configuration/save", nil)
}

func sameIPSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[string]bool, len(a))
	for _, ip := range a {
		set[strings.TrimSpace(ip)] = true
	}
	for _, ip := range b {
		if !set[strings.TrimSpace(ip)] {
			return false
		}
	}
	return true
}

func syncAllDomains(force bool) (*DomainStore, error) {
	store, err := svc.loadDomainStore()
	if err != nil {
		return nil, err
	}

	if !force && !store.AutoRefresh {
		return store, nil
	}
	if strings.TrimSpace(store.Interface) == "" {
		return nil, fmt.Errorf("interface not selected")
	}

	updatedIPs := make(map[string]bool)

	for i := range store.Entries {
		e := &store.Entries[i]
		if e.Disabled || strings.TrimSpace(e.Domain) == "" {
			continue
		}

		ips, err := lookupDomainIPv4(e.Domain)
		if err != nil {
			log.Printf("syncAll lookup error for %s: %v", e.Domain, err)
			continue
		}
		if sameIPSet(e.IPs, ips) {
			continue
		}
		e.IPs = ips
		e.LastLookup = time.Now().UTC().Format(time.RFC3339)

		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				updatedIPs[ip] = true
			}
		}
	}

	for ip := range updatedIPs {
		if err := updateRoutesForIP(store, ip); err != nil {
			log.Printf("syncAll updateRoutesForIP error %s: %v", ip, err)
		}
	}

	if err := svc.saveDomainStore(store); err != nil {
		return nil, err
	}
	if err := svc.client.RciPost("system/configuration/save", nil); err != nil {
		return nil, err
	}

	return store, nil
}