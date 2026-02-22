// internal/routes/store.go
package routes

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"keenetic-go-vpn/internal/config"
	"keenetic-go-vpn/keenetic"
)

type Service struct {
	cfg       config.Config
	client    *keenetic.Client
	storePath string
	storeMu   sync.Mutex
}

var svc *Service

func InitStore(cfg config.Config, client *keenetic.Client) {
	svc = &Service{
		cfg:       cfg,
		client:    client,
		storePath: cfg.DomainStorePath,
	}
}

func (s *Service) loadDomainStore() (*DomainStore, error) {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	data, err := os.ReadFile(s.storePath)
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

func (s *Service) saveDomainStore(store *DomainStore) error {
	s.storeMu.Lock()
	defer s.storeMu.Unlock()

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.storePath, data, 0o644)
}

func (s *Service) findDomainEntry(store *DomainStore, domain string) (*DomainRouteEntry, int) {
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
	sort.Strings(res)
	return res, nil
}