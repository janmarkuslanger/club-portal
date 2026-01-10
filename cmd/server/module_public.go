package main

import (
	"net/http"

	"github.com/janmarkuslanger/club-portal/internal/auth"
	"github.com/janmarkuslanger/graft/module"
	"github.com/janmarkuslanger/graft/router"
)

type publicDeps struct {
	Sessions  *auth.Manager
	Templates templates
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
	if _, ok := sessionUserID(deps.Sessions, ctx.Request); ok {
		http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/login", http.StatusSeeOther)
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
