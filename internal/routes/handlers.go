// internal/routes/handlers.go
package routes

import (
	"net/http"
	"strings"
	"time"

	"keenetic-go-vpn/keenetic"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	client *keenetic.Client
}

func NewHandler(client *keenetic.Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) GetData(c *gin.Context) {
	var ifResp ShowInterfaceResponse
	if err := h.client.RciGet("show/interface", &ifResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type Resp struct {
		Interfaces        []RouterInterface  `json:"interfaces"`
		SelectedInterface string             `json:"selected_interface"`
		Domains           []DomainRouteEntry `json:"domains"`
		AutoRefresh       bool               `json:"auto_refresh"`
	}
	c.JSON(http.StatusOK, Resp{
		Interfaces:        interfaces,
		SelectedInterface: store.Interface,
		Domains:           store.Entries,
		AutoRefresh:       store.AutoRefresh,
	})
}

func (h *Handler) SetInterface(c *gin.Context) {
	var req struct {
		Interface string `json:"interface"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Interface) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	store.Interface = strings.TrimSpace(req.Interface)
	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) SetAutoRefresh(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	store.AutoRefresh = req.Enabled
	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) SyncAll(c *gin.Context) {
	store, err := syncAllDomains(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": store.Entries})
}

func (h *Handler) AddDomain(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(store.Interface) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interface not selected"})
		return
	}

	ips, err := lookupDomainIPv4(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entry, idx := svc.findDomainEntry(store, domain)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

func (h *Handler) LookupDomain(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entry, idx := svc.findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	ips, err := lookupDomainIPv4(domain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entry.IPs = ips
	entry.LastLookup = time.Now().UTC().Format(time.RFC3339)
	store.Entries[idx] = *entry

	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

func (h *Handler) EditDomain(c *gin.Context) {
	var req struct {
		Domain string   `json:"domain"`
		IPs    []string `json:"ips"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entry, idx := svc.findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
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

	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

func (h *Handler) ApplyDomain(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(store.Interface) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interface not selected"})
		return
	}

	entry, idx := svc.findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	if err := applyRoutesForDomain(store, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	store.Entries[idx] = *entry
	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

func (h *Handler) DeleteDomain(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entry, idx := svc.findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	entry.Disabled = true
	if err := removeRoutesForDomain(store, entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	store.Entries = append(store.Entries[:idx], store.Entries[idx+1:]...)
	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) SetDomainActive(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
		Active bool   `json:"active"`
	}
	if err := c.BindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}
	domain := strings.ToLower(strings.TrimSpace(req.Domain))

	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	entry, idx := svc.findDomainEntry(store, domain)
	if entry == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	if !req.Active {
		entry.Disabled = true
		if err := removeRoutesForDomain(store, entry); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		entry.Disabled = false
		if err := applyRoutesForDomain(store, entry); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	store.Entries[idx] = *entry
	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

func (h *Handler) ActivateAll(c *gin.Context) {
	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(store.Interface) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "interface not selected"})
		return
	}

	for i := range store.Entries {
		e := &store.Entries[i]
		e.Disabled = false
		if err := applyRoutesForDomain(store, e); err != nil {
			// continue but log
			continue
		}
	}

	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": store.Entries})
}

func (h *Handler) DeactivateAll(c *gin.Context) {
	store, err := svc.loadDomainStore()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for i := range store.Entries {
		e := &store.Entries[i]
		e.Disabled = true
		if err := removeRoutesForDomain(store, e); err != nil {
			// continue but log
			continue
		}
	}

	if err := svc.saveDomainStore(store); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": store.Entries})
}