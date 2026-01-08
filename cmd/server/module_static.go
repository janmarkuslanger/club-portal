package main

import "github.com/janmarkuslanger/graft/router"

type staticModule struct {
	AdminAssetsDir string
	SiteAssetsDir  string
	ClubsDir       string
}

func (m staticModule) BuildRoutes(r router.Router) {
	if m.AdminAssetsDir != "" {
		r.Static("/admin-assets", m.AdminAssetsDir)
	}
	if m.SiteAssetsDir != "" {
		r.Static("/assets", m.SiteAssetsDir)
	}
	if m.ClubsDir != "" {
		r.Static("/clubs", m.ClubsDir)
	}
}
