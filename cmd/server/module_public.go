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
	categorySet := make(map[string]string)
	for _, club := range clubs {
		name := strings.TrimSpace(club.Name)
		description := strings.TrimSpace(club.Description)
		city := strings.TrimSpace(club.AddressCity)
		country := strings.TrimSpace(club.AddressCountry)
		rawCategories := store.SplitCategories(club.Categories)
		categories := make([]string, 0, len(rawCategories))
		location := city
		if location == "" {
			location = country
		} else if country != "" {
			location = location + ", " + country
		}

		categorySearch := make([]string, 0, len(rawCategories))
		for _, category := range rawCategories {
			trimmed := strings.TrimSpace(category)
			if trimmed == "" {
				continue
			}
			label := categoryLabelForValue(trimmed)
			categories = append(categories, label)
			lower := strings.ToLower(label)
			categorySearch = append(categorySearch, lower)
			if _, ok := categorySet[lower]; !ok {
				categorySet[lower] = label
			}
		}

		searchText := strings.ToLower(strings.Join([]string{
			name,
			description,
			city,
			country,
			strings.Join(categories, " "),
		}, " "))

		data.Clubs = append(data.Clubs, homeClub{
			Name:           name,
			Slug:           strings.TrimSpace(club.Slug),
			Description:    description,
			Location:       location,
			City:           city,
			Categories:     categories,
			SearchText:     searchText,
			CategorySearch: strings.Join(categorySearch, "|"),
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

	data.Categories = buildHomeCategories(categorySet)

	return data
}

func buildHomeCategories(categorySet map[string]string) []homeCategory {
	used := make(map[string]string, len(categorySet))
	for value, label := range categorySet {
		used[value] = label
	}

	result := make([]homeCategory, 0, len(used))
	known := make(map[string]struct{}, len(categoryOptions))
	for _, option := range categoryOptions {
		label, ok := used[option.Value]
		if !ok {
			continue
		}
		if strings.TrimSpace(label) == "" {
			label = option.Label
		}
		result = append(result, homeCategory{
			Value: option.Value,
			Label: label,
			Icon:  option.Icon,
		})
		known[option.Value] = struct{}{}
	}

	unknown := make([]homeCategory, 0)
	for value, label := range used {
		if _, ok := known[value]; ok {
			continue
		}
		if strings.TrimSpace(label) == "" {
			continue
		}
		unknown = append(unknown, homeCategory{
			Value: value,
			Label: label,
			Icon:  defaultCategoryIcon,
		})
	}
	sort.Slice(unknown, func(i, j int) bool {
		return strings.ToLower(unknown[i].Label) < strings.ToLower(unknown[j].Label)
	})

	return append(result, unknown...)
}
