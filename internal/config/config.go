package config

import (
	"log"
	"os"
)

type Config struct {
	RouterIP   string
	RouterUser string
	RouterPass string
	WebUser    string
	WebPass    string
	WebTTL     string
	HTTPPort   string

	DomainStorePath string
}

func Load() Config {
	var cfg Config

	cfg.RouterIP = os.Getenv("ROUTER_IP")
	cfg.RouterUser = os.Getenv("ROUTER_USER")
	cfg.RouterPass = os.Getenv("ROUTER_PASS")
	cfg.WebUser = os.Getenv("WEB_USER")
	cfg.WebPass = os.Getenv("WEB_PASS")
	cfg.WebTTL = os.Getenv("WEB_SESSION_TTL")
	cfg.HTTPPort = os.Getenv("HTTP_PORT")
	cfg.DomainStorePath = os.Getenv("DOMAIN_STORE_PATH")

	if cfg.RouterIP == "" {
		cfg.RouterIP = "192.168.1.1"
	}
	if cfg.WebUser == "" {
		log.Fatal("WEB_USER required")
	}
	if cfg.WebPass == "" {
		log.Fatal("WEB_PASS required")
	}
	if cfg.WebTTL == "" {
		log.Fatal("WEB_SESSION_TTL required")
	}
	if cfg.HTTPPort == "" {
		cfg.HTTPPort = "8000"
	}
	if cfg.DomainStorePath == "" {
		cfg.DomainStorePath = "/data/domains.json"
	}

	return cfg
}