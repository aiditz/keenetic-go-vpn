package http

import (
	"fmt"
	"net/http"

	"keenetic-go-vpn/internal/auth"
	"keenetic-go-vpn/internal/config"
	dev "keenetic-go-vpn/internal/devices"
	rt "keenetic-go-vpn/internal/routes"
	"keenetic-go-vpn/keenetic"

	"github.com/gin-gonic/gin"
)

func NewServer(cfg config.Config, routerClient *keenetic.Client) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Delims("[[", "]]")
	r.Static("/assets", "./static/assets")
	r.LoadHTMLFiles("./static/index.html")

	// Session manager + auth handlers
	sessMgr := auth.NewManager(cfg)
	authHandler := auth.NewHandler(sessMgr)

	// Public routes
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"User": cfg.WebUser,
			"Host": cfg.RouterIP,
			"TTL":  cfg.WebTTL,
		})
	})

	publicAPI := r.Group("/api")
	{
		publicAPI.POST("/login", authHandler.Login)
		publicAPI.POST("/logout", authHandler.Logout)
	}

	// Protected API (requires session cookie)
	protected := r.Group("/api", sessMgr.Middleware())
	{
		devicesHandler := dev.NewHandler(routerClient)
		protected.GET("/data", devicesHandler.GetDevices)
		protected.POST("/policy", devicesHandler.SetPolicy)
		protected.POST("/static_ip", devicesHandler.SetStaticIP)

		routesHandler := rt.NewHandler(routerClient)
		routes := protected.Group("/routes")
		{
			routes.GET("/data", routesHandler.GetData)
			routes.POST("/interface", routesHandler.SetInterface)
			routes.POST("/auto_refresh", routesHandler.SetAutoRefresh)
			routes.POST("/sync_all", routesHandler.SyncAll)

			routes.POST("/domain/add", routesHandler.AddDomain)
			routes.POST("/domain/lookup", routesHandler.LookupDomain)
			routes.POST("/domain/edit", routesHandler.EditDomain)
			routes.POST("/domain/apply", routesHandler.ApplyDomain)
			routes.POST("/domain/delete", routesHandler.DeleteDomain)
			routes.POST("/domain/active", routesHandler.SetDomainActive)

			routes.POST("/activate_all", routesHandler.ActivateAll)
			routes.POST("/deactivate_all", routesHandler.DeactivateAll)
		}
	}

	fmt.Printf("HTTP server ready on :%s\n", cfg.HTTPPort)
	return r
}