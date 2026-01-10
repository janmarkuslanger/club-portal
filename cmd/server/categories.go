package main

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/janmarkuslanger/club-portal/internal/store"
)

type categoryOption struct {
	Value string
	Label string
	Icon  template.HTML
}

var categoryOptions = []categoryOption{
	{
		Value: "fitness",
		Label: "Fitness",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 9v6M20 9v6M7 12h10M6 10h1v4H6zM17 10h1v4h-1z"/></svg>`),
	},
	{
		Value: "kampfsport",
		Label: "Kampfsport",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M12 3l7 3v6c0 4-3 7-7 9-4-2-7-5-7-9V6l7-3z"/></svg>`),
	},
	{
		Value: "teamsport",
		Label: "Teamsport",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="9" cy="8" r="3"/><circle cx="17" cy="9" r="2.5"/><path d="M4 20c0-3 3-5 5-5s5 2 5 5"/><path d="M14 19c.3-2 2-3.5 4-3.5 1.6 0 3 1 3.5 2.5"/></svg>`),
	},
	{
		Value: "yoga",
		Label: "Yoga",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="5" r="2"/><path d="M7 20c3-2 7-2 10 0"/><path d="M5 13c2.5-2 5-3 7-3s4.5 1 7 3"/><path d="M12 7v4"/></svg>`),
	},
	{
		Value: "tanz",
		Label: "Tanz",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M9 18c3 0 5-2 5-5V4"/><circle cx="16" cy="4" r="2"/><path d="M7 20c-2 0-3-1-3-3 0-2 1-3 3-3 3 0 5-2 5-5"/></svg>`),
	},
	{
		Value: "outdoor",
		Label: "Outdoor",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 20l6-10 4 6 2-3 6 7"/><path d="M9 10l3-5 4 7"/></svg>`),
	},
	{
		Value: "schwimmen",
		Label: "Schwimmen",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 18c2 2 4 2 6 0 2 2 4 2 6 0 2 2 4 2 6 0"/><path d="M6 12c2 2 4 2 6 0 2 2 4 2 6 0"/><circle cx="8" cy="7" r="2"/></svg>`),
	},
	{
		Value: "gesundheit",
		Label: "Gesundheit",
		Icon:  template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M20 8c0-2-1.5-4-4-4-2 0-3.5 1.5-4 3-0.5-1.5-2-3-4-3-2.5 0-4 2-4 4 0 6 8 10 8 10s8-4 8-10z"/></svg>`),
	},
}

var defaultCategoryIcon = template.HTML(`<svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M3 12l9-9 9 9-9 9-9-9z"/><path d="M12 7v10"/></svg>`)

func categoryOptionsList() []categoryOption {
	return categoryOptions
}

func categoriesFromForm(r *http.Request) string {
	raw := append([]string{}, r.Form["category"]...)
	selected := make([]string, 0, len(raw))
	for _, value := range raw {
		label := categoryLabelForValue(value)
		if label != "" {
			selected = append(selected, label)
		}
	}
	custom := strings.TrimSpace(r.FormValue("category_custom"))
	if custom != "" {
		selected = append(selected, custom)
	}
	if len(selected) == 0 {
		return ""
	}
	return store.NormalizeCategories(strings.Join(selected, ", "))
}

func categoryLabelForValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	for _, option := range categoryOptions {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}

func categorySelection(categories string) (map[string]bool, string) {
	items := store.SplitCategories(categories)
	selection := make(map[string]bool, len(categoryOptions))
	known := make(map[string]struct{}, len(categoryOptions))
	for _, opt := range categoryOptions {
		known[opt.Value] = struct{}{}
	}

	custom := make([]string, 0)
	for _, item := range items {
		lower := strings.ToLower(item)
		if _, ok := known[lower]; ok {
			selection[lower] = true
			continue
		}
		custom = append(custom, item)
	}

	return selection, strings.Join(custom, ", ")
}
