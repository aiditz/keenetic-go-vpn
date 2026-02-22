package main

import (
	"fmt"
	"log"

	"keenetic-go-vpn/internal/config"
	"keenetic-go-vpn/internal/http"
	"keenetic-go-vpn/internal/router"
	"keenetic-go-vpn/internal/routes"
)

func main() {
	cfg := config.Load()

	routerClient, err := router.NewClient(cfg)
	if err != nil {
		log.Fatalf("router client init error: %v", err)
	}

	routes.InitStore(cfg, routerClient)
	routes.ScheduleAutoRefresh() // ночной syncAll (учитывает AutoRefresh)

	engine := http.NewServer(cfg, routerClient)

	fmt.Printf("Starting Keenetic Go on port %s...\n", cfg.HTTPPort)
	if err := engine.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatalf("server error: %v", err)
	}
}