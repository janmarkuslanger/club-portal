package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/janmarkuslanger/club-portal/internal/auth"
	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/graft/module"
	"github.com/janmarkuslanger/graft/router"
)

type authDeps struct {
	Store        *store.Store
	Sessions     *auth.Manager
	Templates    templates
	CookieSecure bool
}

func authModule(deps authDeps) *module.Module[authDeps] {
	mod := &module.Module[authDeps]{
		Name:     "auth",
		BasePath: "",
		Deps:     deps,
		Routes: []module.Route[authDeps]{
			{Method: http.MethodPost, Path: "/login", Handler: handleLoginSubmit},
			{Method: http.MethodPost, Path: "/register", Handler: handleRegisterSubmit},
		},
	}
	return mod
}

func handleLoginSubmit(ctx router.Context, deps authDeps) {
	if err := ctx.Request.ParseForm(); err != nil {
		http.Error(ctx.Writer, "invalid form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(ctx.Request.FormValue("email"))
	password := ctx.Request.FormValue("password")

	user, err := deps.Store.Authenticate(email, password)
	if err != nil {
		data := loginData{
			AppName: appName(),
			Title:   "Login",
			Error:   "Login fehlgeschlagen. Bitte pruefe deine Daten.",
			Email:   email,
		}
		renderTemplate(ctx.Writer, deps.Templates.login, data)
		return
	}

	sessionToken := deps.Sessions.Create(user.ID)
	setSessionCookie(ctx.Writer, sessionToken, deps.CookieSecure)

	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
}

func handleRegisterSubmit(ctx router.Context, deps authDeps) {
	if err := ctx.Request.ParseForm(); err != nil {
		http.Error(ctx.Writer, "invalid form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(ctx.Request.FormValue("email"))
	password := ctx.Request.FormValue("password")

	user, err := deps.Store.CreateUser(email, password)
	if err != nil {
		msg := "Registrierung fehlgeschlagen."
		switch {
		case errors.Is(err, store.ErrEmailExists):
			msg = "Diese E-Mail ist bereits registriert."
		case errors.Is(err, store.ErrPasswordTooShort):
			msg = "Passwort ist zu kurz."
		}
		data := registerData{
			AppName: appName(),
			Title:   "Registrieren",
			Error:   msg,
			Email:   email,
		}
		renderTemplate(ctx.Writer, deps.Templates.register, data)
		return
	}

	sessionToken := deps.Sessions.Create(user.ID)
	setSessionCookie(ctx.Writer, sessionToken, deps.CookieSecure)

	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
}
