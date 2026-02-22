package devices

import (
	"net/http"

	"keenetic-go-vpn/keenetic"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc    *Service
	client *keenetic.Client
}

func NewHandler(c *keenetic.Client) *Handler {
	return &Handler{
		svc:    NewService(c),
		client: c,
	}
}

func (h *Handler) GetDevices(c *gin.Context) {
	devices, policies, err := h.svc.FetchRouterData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devices, "policies": policies})
}

func (h *Handler) SetPolicy(c *gin.Context) {
	var req struct {
		MAC      string `json:"mac"`
		PolicyID string `json:"policy_id"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	if req.PolicyID == "_DENY_" {
		if err := h.client.RciPost("ip/hotspot/host", map[string]string{
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
		if err := h.client.RciPost("ip/hotspot/host", payload); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := h.client.RciPost("ip/hotspot/host", map[string]string{
			"mac":    req.MAC,
			"access": "permit",
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := h.client.RciPost("ip/hotspot/host", map[string]interface{}{
			"mac":    req.MAC,
			"policy": req.PolicyID,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	_ = h.client.RciPost("system/configuration/save", nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) SetStaticIP(c *gin.Context) {
	var req struct {
		MAC string `json:"mac"`
		IP  string `json:"ip"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
		return
	}

	if req.IP == "" {
		if err := h.client.RciPost("ip/dhcp/host", map[string]interface{}{
			"mac": req.MAC,
			"no":  true,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := h.client.RciPost("ip/dhcp/host", map[string]string{
			"mac": req.MAC,
			"ip":  req.IP,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	_ = h.client.RciPost("system/configuration/save", nil)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}