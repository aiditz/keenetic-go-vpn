// internal/routes/scheduler.go
package routes

import (
	"log"
	"time"
)

func ScheduleAutoRefresh() {
	go func() {
		for {
			now := time.Now().UTC()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
			time.Sleep(next.Sub(now))

			store, err := svc.loadDomainStore()
			if err != nil {
				log.Printf("Auto sync: load error: %v", err)
				continue
			}
			if !store.AutoRefresh {
				continue
			}

			if _, err := syncAllDomains(false); err != nil {
				log.Printf("Auto syncAll error: %v", err)
			}
		}
	}()
}