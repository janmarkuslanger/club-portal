package main

import (
	"log"

	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/graft/router"
)

type seedModule struct {
	Store *store.Store
}

func (m seedModule) BuildRoutes(r router.Router) {}

func (m seedModule) OnStart() {
	if m.Store == nil {
		return
	}

	seed, created, err := m.Store.EnsureExampleClub()
	if err != nil {
		log.Fatal(err)
	}
	if created {
		log.Printf("seeded example club %q (login: %s / %s)", seed.Club.Name, seed.Email, seed.Password)
	}
}
