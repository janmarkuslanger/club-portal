package main

import (
	"errors"
	"net/http"

	"github.com/janmarkuslanger/club-portal/internal/auth"
	"github.com/janmarkuslanger/club-portal/internal/site"
	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/graft/module"
	"github.com/janmarkuslanger/graft/router"
)

type adminDeps struct {
	Store        *store.Store
	Sessions     *auth.Manager
	Templates    templates
	BuildOptions site.BuildOptions
	CookieSecure bool
}

func adminModule(deps adminDeps) *module.Module[adminDeps] {
	mod := &module.Module[adminDeps]{
		Name:        "admin",
		BasePath:    "",
		Deps:        deps,
		Middlewares: []router.Middleware{requireAuth(deps.Sessions)},
		Routes: []module.Route[adminDeps]{
			{Method: http.MethodGet, Path: "/admin", Handler: handleDashboard},
			{Method: http.MethodPost, Path: "/admin/club", Handler: handleClubUpdate},
			{Method: http.MethodPost, Path: "/admin/build", Handler: handleBuild},
			{Method: http.MethodPost, Path: "/logout", Handler: handleLogout},
		},
	}
	return mod
}

func handleDashboard(ctx router.Context, deps adminDeps) {
	userID, ok := sessionUserID(deps.Sessions, ctx.Request)
	if !ok {
		http.Redirect(ctx.Writer, ctx.Request, "/login", http.StatusSeeOther)
		return
	}

	club, hasClub := deps.Store.GetClubByOwner(userID)
	info := ""
	if ctx.Request.URL.Query().Get("saved") == "1" {
		info = "Club gespeichert."
	}
	if ctx.Request.URL.Query().Get("build") == "1" {
		info = "Static site aktualisiert."
	}

	data := dashboardData{
		Title:           "Dashboard",
		Info:            info,
		ClubName:        club.Name,
		ClubDescription: club.Description,
		ClubSlug:        club.Slug,
	}
	if hasClub {
		data.PreviewPath = "/clubs/" + club.Slug + "/"
	}

	renderTemplate(ctx.Writer, deps.Templates.dashboard, data)
}

func handleClubUpdate(ctx router.Context, deps adminDeps) {
	userID, ok := sessionUserID(deps.Sessions, ctx.Request)
	if !ok {
		http.Redirect(ctx.Writer, ctx.Request, "/login", http.StatusSeeOther)
		return
	}

	existingClub, hasClub := deps.Store.GetClubByOwner(userID)

	if err := ctx.Request.ParseForm(); err != nil {
		http.Error(ctx.Writer, "invalid form", http.StatusBadRequest)
		return
	}

	name := ctx.Request.FormValue("name")
	description := ctx.Request.FormValue("description")

	_, err := deps.Store.UpsertClub(userID, name, description)
	if err != nil {
		data := dashboardData{
			Title:           "Dashboard",
			Error:           clubErrorMessage(err),
			ClubName:        name,
			ClubDescription: description,
			ClubSlug:        existingClub.Slug,
		}
		if hasClub && existingClub.Slug != "" {
			data.PreviewPath = "/clubs/" + existingClub.Slug + "/"
		}
		renderTemplate(ctx.Writer, deps.Templates.dashboard, data)
		return
	}

	http.Redirect(ctx.Writer, ctx.Request, "/admin?saved=1", http.StatusSeeOther)
}

func handleBuild(ctx router.Context, deps adminDeps) {
	userID, ok := sessionUserID(deps.Sessions, ctx.Request)
	if !ok {
		http.Redirect(ctx.Writer, ctx.Request, "/login", http.StatusSeeOther)
		return
	}

	clubs := deps.Store.AllClubs()
	if err := site.Build(clubs, deps.BuildOptions); err != nil {
		club, hasClub := deps.Store.GetClubByOwner(userID)
		data := dashboardData{
			Title:           "Dashboard",
			Error:           "Build fehlgeschlagen. Bitte erneut versuchen.",
			ClubName:        club.Name,
			ClubDescription: club.Description,
			ClubSlug:        club.Slug,
		}
		if hasClub && club.Slug != "" {
			data.PreviewPath = "/clubs/" + club.Slug + "/"
		}
		renderTemplate(ctx.Writer, deps.Templates.dashboard, data)
		return
	}

	http.Redirect(ctx.Writer, ctx.Request, "/admin?build=1", http.StatusSeeOther)
}

func handleLogout(ctx router.Context, deps adminDeps) {
	cookie, err := ctx.Request.Cookie(sessionCookieName)
	if err == nil {
		deps.Sessions.Delete(cookie.Value)
	}

	clearSessionCookie(ctx.Writer, deps.CookieSecure)
	http.Redirect(ctx.Writer, ctx.Request, "/login", http.StatusSeeOther)
}

func requireAuth(sessions *auth.Manager) router.Middleware {
	return func(ctx router.Context, next router.HandlerFunc) {
		if _, ok := sessionUserID(sessions, ctx.Request); !ok {
			http.Redirect(ctx.Writer, ctx.Request, "/login", http.StatusSeeOther)
			return
		}
		next(ctx)
	}
}

func clubErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, store.ErrNameRequired) {
		return "Bitte einen Clubnamen angeben."
	}
	return "Speichern fehlgeschlagen."
}
