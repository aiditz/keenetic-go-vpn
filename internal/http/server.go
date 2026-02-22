package http

import (
	"fmt"
	"net/http"

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
		devicesHandler := dev.NewHandler(routerClient)
		api.GET("/data", devicesHandler.GetDevices)
		api.POST("/policy", devicesHandler.SetPolicy)
		api.POST("/static_ip", devicesHandler.SetStaticIP)

		routesHandler := rt.NewHandler(routerClient)
		routes := api.Group("/routes")
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