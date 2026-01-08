package main

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/janmarkuslanger/club-portal/internal/auth"
	"github.com/janmarkuslanger/club-portal/internal/site"
	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/graft/graft"
	"github.com/janmarkuslanger/graft/module"
	"github.com/janmarkuslanger/graft/router"
)

const sessionCookieName = "club_portal_session"

const (
	defaultDataPath    = "data/store.json"
	defaultOutputDir   = "public"
	defaultTemplateDir = "templates/site"
	defaultAssetDir    = "static/site"
)

type appDeps struct {
	Store        *store.Store
	Sessions     *auth.Manager
	Templates    templates
	BuildOptions site.BuildOptions
	CookieSecure bool
}

type templates struct {
	login     *template.Template
	register  *template.Template
	dashboard *template.Template
}

type loginData struct {
	Title string
	Error string
	Email string
}

type registerData struct {
	Title string
	Error string
	Email string
}

type dashboardData struct {
	Title           string
	Error           string
	Info            string
	ClubName        string
	ClubDescription string
	ClubSlug        string
	PreviewPath     string
}

func main() {
	dataPath := envOrDefault("DATA_PATH", defaultDataPath)
	outputDir := envOrDefault("OUTPUT_DIR", defaultOutputDir)
	templateDir := envOrDefault("TEMPLATE_DIR", defaultTemplateDir)
	assetDir := envOrDefault("ASSET_DIR", defaultAssetDir)

	storeInstance, err := store.NewStore(dataPath)
	if err != nil {
		log.Fatal(err)
	}

	sessions := auth.NewManager(envDuration("SESSION_TTL", 24*time.Hour))
	cookieSecure := envBool("COOKIE_SECURE", false)

	tmpls, err := loadTemplates(filepath.Join("templates", "admin"))
	if err != nil {
		log.Fatal(err)
	}

	deps := appDeps{
		Store:     storeInstance,
		Sessions:  sessions,
		Templates: tmpls,
		BuildOptions: site.BuildOptions{
			OutputDir:   outputDir,
			TemplateDir: templateDir,
			AssetDir:    assetDir,
		},
		CookieSecure: cookieSecure,
	}

	app := graft.New()
	app.UseModule(staticModule{
		AdminAssetsDir: filepath.Join("static", "admin"),
		SiteAssetsDir:  filepath.Join(outputDir, "assets"),
		ClubsDir:       filepath.Join(outputDir, "clubs"),
	})

	app.UseModule(publicModule(deps))
	app.UseModule(adminModule(deps))

	log.Println("club-portal server running on :8080")
	app.Run()
}

func publicModule(deps appDeps) *module.Module[appDeps] {
	mod := &module.Module[appDeps]{
		Name:     "public",
		BasePath: "",
		Deps:     deps,
		Routes: []module.Route[appDeps]{
			{Method: http.MethodGet, Path: "/", Handler: handleHome},
			{Method: http.MethodGet, Path: "/login", Handler: handleLoginForm},
			{Method: http.MethodPost, Path: "/login", Handler: handleLoginSubmit},
			{Method: http.MethodGet, Path: "/register", Handler: handleRegisterForm},
			{Method: http.MethodPost, Path: "/register", Handler: handleRegisterSubmit},
		},
	}
	return mod
}

func adminModule(deps appDeps) *module.Module[appDeps] {
	mod := &module.Module[appDeps]{
		Name:        "admin",
		BasePath:    "",
		Deps:        deps,
		Middlewares: []router.Middleware{requireAuth(deps.Sessions)},
		Routes: []module.Route[appDeps]{
			{Method: http.MethodGet, Path: "/admin", Handler: handleDashboard},
			{Method: http.MethodPost, Path: "/admin/club", Handler: handleClubUpdate},
			{Method: http.MethodPost, Path: "/admin/build", Handler: handleBuild},
			{Method: http.MethodPost, Path: "/logout", Handler: handleLogout},
		},
	}
	return mod
}

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

func handleHome(ctx router.Context, deps appDeps) {
	if _, ok := sessionUserID(deps.Sessions, ctx.Request); ok {
		http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/login", http.StatusSeeOther)
}

func handleLoginForm(ctx router.Context, deps appDeps) {
	if _, ok := sessionUserID(deps.Sessions, ctx.Request); ok {
		http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
		return
	}

	data := loginData{
		Title: "Login",
		Error: errorMessage(ctx.Request.URL.Query().Get("error")),
		Email: ctx.Request.URL.Query().Get("email"),
	}

	renderTemplate(ctx.Writer, deps.Templates.login, data)
}

func handleLoginSubmit(ctx router.Context, deps appDeps) {
	if err := ctx.Request.ParseForm(); err != nil {
		http.Error(ctx.Writer, "invalid form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(ctx.Request.FormValue("email"))
	password := ctx.Request.FormValue("password")

	user, err := deps.Store.Authenticate(email, password)
	if err != nil {
		data := loginData{
			Title: "Login",
			Error: "Login fehlgeschlagen. Bitte pruefe deine Daten.",
			Email: email,
		}
		renderTemplate(ctx.Writer, deps.Templates.login, data)
		return
	}

	sessionToken := deps.Sessions.Create(user.ID)
	setSessionCookie(ctx.Writer, sessionToken, deps.CookieSecure)

	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
}

func handleRegisterForm(ctx router.Context, deps appDeps) {
	if _, ok := sessionUserID(deps.Sessions, ctx.Request); ok {
		http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
		return
	}

	data := registerData{
		Title: "Registrieren",
		Error: errorMessage(ctx.Request.URL.Query().Get("error")),
		Email: ctx.Request.URL.Query().Get("email"),
	}

	renderTemplate(ctx.Writer, deps.Templates.register, data)
}

func handleRegisterSubmit(ctx router.Context, deps appDeps) {
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
			Title: "Registrieren",
			Error: msg,
			Email: email,
		}
		renderTemplate(ctx.Writer, deps.Templates.register, data)
		return
	}

	sessionToken := deps.Sessions.Create(user.ID)
	setSessionCookie(ctx.Writer, sessionToken, deps.CookieSecure)

	http.Redirect(ctx.Writer, ctx.Request, "/admin", http.StatusSeeOther)
}

func handleDashboard(ctx router.Context, deps appDeps) {
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

func handleClubUpdate(ctx router.Context, deps appDeps) {
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

func handleBuild(ctx router.Context, deps appDeps) {
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

func handleLogout(ctx router.Context, deps appDeps) {
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

func sessionUserID(sessions *auth.Manager, r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return sessions.Get(cookie.Value)
}

func setSessionCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   -1,
	})
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func loadTemplates(dir string) (templates, error) {
	login, err := template.ParseFiles(filepath.Join(dir, "login.html"))
	if err != nil {
		return templates{}, err
	}
	register, err := template.ParseFiles(filepath.Join(dir, "register.html"))
	if err != nil {
		return templates{}, err
	}
	dashboard, err := template.ParseFiles(filepath.Join(dir, "dashboard.html"))
	if err != nil {
		return templates{}, err
	}

	return templates{
		login:     login,
		register:  register,
		dashboard: dashboard,
	}, nil
}

func errorMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}
	return msg
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

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
