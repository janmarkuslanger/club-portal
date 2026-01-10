package main

import (
	"net/http"
	"sort"
	"strings"

	"github.com/janmarkuslanger/club-portal/internal/auth"
	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/graft/module"
	"github.com/janmarkuslanger/graft/router"
)

type publicDeps struct {
	Sessions  *auth.Manager
	Templates templates
	Store     *store.Store
}

func publicModule(deps publicDeps) *module.Module[publicDeps] {
	mod := &module.Module[publicDeps]{
		Name:     "public",
		BasePath: "",
		Deps:     deps,
		Routes: []module.Route[publicDeps]{
			{Method: http.MethodGet, Path: "/", Handler: handleHome},
			{Method: http.MethodGet, Path: "/login", Handler: handleLoginForm},
			{Method: http.MethodGet, Path: "/register", Handler: handleRegisterForm},
		},
	}
	return mod
}

func handleHome(ctx router.Context, deps publicDeps) {
	clubs := deps.Store.AllClubs()
	data := homeDataFromClubs(clubs)
	renderTemplate(ctx.Writer, deps.Templates.home, data)
}

func handleLoginForm(ctx router.Context, deps publicDeps) {
	if _, ok := sessionUserID(deps.Sessions, ctx.Request); ok {
		http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
		return
	}

	data := loginData{
		AppName: appName(),
		Title:   "Login",
		Error:   errorMessage(ctx.Request.URL.Query().Get("error")),
		Email:   ctx.Request.URL.Query().Get("email"),
	}

	renderTemplate(ctx.Writer, deps.Templates.login, data)
}

func handleRegisterForm(ctx router.Context, deps publicDeps) {
	if _, ok := sessionUserID(deps.Sessions, ctx.Request); ok {
		http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
		return
	}

	data := registerData{
		AppName: appName(),
		Title:   "Registrieren",
		Error:   errorMessage(ctx.Request.URL.Query().Get("error")),
		Email:   ctx.Request.URL.Query().Get("email"),
	}

	renderTemplate(ctx.Writer, deps.Templates.register, data)
}

func homeDataFromClubs(clubs []store.Club) homeData {
	data := homeData{
		AppName:   appName(),
		Title:     "Start",
		ClubCount: len(clubs),
		Clubs:     make([]homeClub, 0, len(clubs)),
	}

	citySet := make(map[string]struct{})
	for _, club := range clubs {
		name := strings.TrimSpace(club.Name)
		description := strings.TrimSpace(club.Description)
		city := strings.TrimSpace(club.AddressCity)
		country := strings.TrimSpace(club.AddressCountry)
		location := city
		if location == "" {
			location = country
		} else if country != "" {
			location = location + ", " + country
		}

		searchText := strings.ToLower(strings.Join([]string{
			name,
			description,
			city,
			country,
		}, " "))

		data.Clubs = append(data.Clubs, homeClub{
			Name:        name,
			Slug:        strings.TrimSpace(club.Slug),
			Description: description,
			Location:    location,
			City:        city,
			SearchText:  searchText,
		})

		if city != "" {
			citySet[city] = struct{}{}
		}
	}

	data.Cities = make([]string, 0, len(citySet))
	for city := range citySet {
		data.Cities = append(data.Cities, city)
	}
	sort.Strings(data.Cities)

	return data
}
