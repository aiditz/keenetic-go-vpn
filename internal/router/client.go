package router

import (
	"log"

	"keenetic-go-vpn/internal/config"
	"keenetic-go-vpn/keenetic"
)

func NewClient(cfg config.Config) (*keenetic.Client, error) {
	client, err := keenetic.NewClient(cfg.RouterIP, cfg.RouterUser, cfg.RouterPass)
	if err != nil {
		return nil, err
	}
	if err := client.Login(); err != nil {
		log.Printf("Warning: initial login failed: %v", err)
	}
	return client, nil
}