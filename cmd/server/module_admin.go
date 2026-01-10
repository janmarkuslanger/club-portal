package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/janmarkuslanger/club-portal/internal/auth"
	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/graft/module"
	"github.com/janmarkuslanger/graft/router"
)

const courseExtraRows = 3

type adminDeps struct {
	Store         *store.Store
	Sessions      *auth.Manager
	Templates     templates
	BuildDebounce time.Duration
	CookieSecure  bool
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

	data := dashboardDataFromClub(club, hasClub)
	data.Title = "Dashboard"
	data.Info = info

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

	update := store.ClubUpdate{
		Name:           ctx.Request.FormValue("name"),
		Description:    ctx.Request.FormValue("description"),
		Categories:     categoriesFromForm(ctx.Request),
		ContactName:    ctx.Request.FormValue("contact_name"),
		ContactRole:    ctx.Request.FormValue("contact_role"),
		ContactEmail:   ctx.Request.FormValue("contact_email"),
		ContactPhone:   ctx.Request.FormValue("contact_phone"),
		ContactWebsite: ctx.Request.FormValue("contact_website"),
		AddressLine1:   ctx.Request.FormValue("address_line1"),
		AddressLine2:   ctx.Request.FormValue("address_line2"),
		AddressPostal:  ctx.Request.FormValue("address_postal"),
		AddressCity:    ctx.Request.FormValue("address_city"),
		AddressCountry: ctx.Request.FormValue("address_country"),
	}

	club, err := deps.Store.UpsertClub(userID, update)
	if err != nil {
		data := dashboardDataFromForm(ctx.Request, existingClub.Slug)
		data.Title = "Dashboard"
		data.Error = clubErrorMessage(err)
		if hasClub && existingClub.Slug != "" {
			data.PreviewPath = "/clubs/" + existingClub.Slug + "/"
		}
		renderTemplate(ctx.Writer, deps.Templates.dashboard, data)
		return
	}

	openingInputs := openingInputsFromForm(ctx.Request)
	if err := deps.Store.ReplaceOpeningHours(club.ID, openingInputs); err != nil {
		data := dashboardDataFromForm(ctx.Request, club.Slug)
		data.Title = "Dashboard"
		data.Error = "Speichern fehlgeschlagen."
		data.PreviewPath = "/clubs/" + club.Slug + "/"
		renderTemplate(ctx.Writer, deps.Templates.dashboard, data)
		return
	}

	courseInputs := courseInputsFromForm(ctx.Request)
	if err := deps.Store.ReplaceCourses(club.ID, courseInputs); err != nil {
		data := dashboardDataFromForm(ctx.Request, club.Slug)
		data.Title = "Dashboard"
		data.Error = "Speichern fehlgeschlagen."
		data.PreviewPath = "/clubs/" + club.Slug + "/"
		renderTemplate(ctx.Writer, deps.Templates.dashboard, data)
		return
	}

	if err := deps.Store.EnqueueBuildTask(deps.BuildDebounce); err != nil {
		log.Printf("failed to enqueue build task: %v", err)
	}

	http.Redirect(ctx.Writer, ctx.Request, "/admin?saved=1", http.StatusSeeOther)
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

func dashboardDataFromClub(club store.Club, hasClub bool) dashboardData {
	data := dashboardData{
		AppName:         appName(),
		ClubName:        club.Name,
		ClubDescription: club.Description,
		ClubCategories:  club.Categories,
		CategoryOptions: categoryOptionsList(),
		ClubSlug:        club.Slug,
		ContactName:     club.ContactName,
		ContactRole:     club.ContactRole,
		ContactEmail:    club.ContactEmail,
		ContactPhone:    club.ContactPhone,
		ContactWebsite:  club.ContactWebsite,
		AddressLine1:    club.AddressLine1,
		AddressLine2:    club.AddressLine2,
		AddressPostal:   club.AddressPostal,
		AddressCity:     club.AddressCity,
		AddressCountry:  club.AddressCountry,
		OpeningHours:    buildOpeningRows(club.OpeningHours),
		Courses:         buildCourseRows(club.Courses),
	}
	data.CategorySelection, data.CategoryCustom = categorySelection(club.Categories)
	if hasClub && club.Slug != "" {
		data.PreviewPath = "/clubs/" + club.Slug + "/"
	}
	return data
}

func dashboardDataFromForm(r *http.Request, clubSlug string) dashboardData {
	categories := categoriesFromForm(r)
	data := dashboardData{
		AppName:         appName(),
		ClubName:        r.FormValue("name"),
		ClubDescription: r.FormValue("description"),
		ClubCategories:  categories,
		CategoryOptions: categoryOptionsList(),
		ClubSlug:        clubSlug,
		ContactName:     r.FormValue("contact_name"),
		ContactRole:     r.FormValue("contact_role"),
		ContactEmail:    r.FormValue("contact_email"),
		ContactPhone:    r.FormValue("contact_phone"),
		ContactWebsite:  r.FormValue("contact_website"),
		AddressLine1:    r.FormValue("address_line1"),
		AddressLine2:    r.FormValue("address_line2"),
		AddressPostal:   r.FormValue("address_postal"),
		AddressCity:     r.FormValue("address_city"),
		AddressCountry:  r.FormValue("address_country"),
		OpeningHours:    openingRowsFromForm(r),
		Courses:         courseRowsFromForm(r),
	}
	data.CategorySelection, data.CategoryCustom = categorySelection(categories)
	if clubSlug != "" {
		data.PreviewPath = "/clubs/" + clubSlug + "/"
	}
	return data
}

func buildOpeningRows(hours []store.OpeningHour) []openingHourRow {
	rows := make([]openingHourRow, 0, 7)
	byDay := make(map[int]store.OpeningHour, len(hours))
	for _, hour := range hours {
		if hour.DayOfWeek < 1 || hour.DayOfWeek > 7 {
			continue
		}
		if _, exists := byDay[hour.DayOfWeek]; !exists {
			byDay[hour.DayOfWeek] = hour
		}
	}

	for day := 1; day <= 7; day++ {
		hour := byDay[day]
		rows = append(rows, openingHourRow{
			Day:      day,
			DayLabel: weekdayLabel(day),
			Open:     hour.OpensAt,
			Close:    hour.ClosesAt,
			Note:     hour.Note,
		})
	}

	return rows
}

func buildCourseRows(courses []store.Course) []courseRow {
	rows := make([]courseRow, 0, len(courses)+courseExtraRows)
	for _, course := range courses {
		rows = append(rows, courseRow{
			Day:         course.DayOfWeek,
			DayLabel:    weekdayLabel(course.DayOfWeek),
			Title:       course.Title,
			Start:       course.StartTime,
			End:         course.EndTime,
			Location:    course.Location,
			Instructor:  course.Instructor,
			Level:       course.Level,
			Description: course.Description,
		})
	}
	rows = append(rows, blankCourseRows(courseExtraRows)...)
	return rows
}

func openingRowsFromForm(r *http.Request) []openingHourRow {
	days := r.Form["opening_day"]
	opens := r.Form["opening_open"]
	closes := r.Form["opening_close"]
	notes := r.Form["opening_note"]

	if len(days) == 0 {
		return buildOpeningRows(nil)
	}

	rows := make([]openingHourRow, 0, len(days))
	for i := 0; i < len(days); i++ {
		day := parseDay(valueAt(days, i), i+1)
		rows = append(rows, openingHourRow{
			Day:      day,
			DayLabel: weekdayLabel(day),
			Open:     valueAt(opens, i),
			Close:    valueAt(closes, i),
			Note:     valueAt(notes, i),
		})
	}

	return rows
}

func courseRowsFromForm(r *http.Request) []courseRow {
	titles := r.Form["course_title"]
	days := r.Form["course_day"]
	starts := r.Form["course_start"]
	ends := r.Form["course_end"]
	locations := r.Form["course_location"]
	instructors := r.Form["course_instructor"]
	levels := r.Form["course_level"]
	descriptions := r.Form["course_description"]

	count := maxLen(titles, days, starts, ends, locations, instructors, levels, descriptions)
	if count == 0 {
		return blankCourseRows(courseExtraRows)
	}

	rows := make([]courseRow, 0, count)
	for i := 0; i < count; i++ {
		day := parseDay(valueAt(days, i), 1)
		rows = append(rows, courseRow{
			Day:         day,
			DayLabel:    weekdayLabel(day),
			Title:       valueAt(titles, i),
			Start:       valueAt(starts, i),
			End:         valueAt(ends, i),
			Location:    valueAt(locations, i),
			Instructor:  valueAt(instructors, i),
			Level:       valueAt(levels, i),
			Description: valueAt(descriptions, i),
		})
	}

	rows = trimTrailingEmptyCourseRows(rows)
	rows = append(rows, blankCourseRows(courseExtraRows)...)
	return rows
}

func openingInputsFromForm(r *http.Request) []store.OpeningHourInput {
	rows := openingRowsFromForm(r)
	inputs := make([]store.OpeningHourInput, 0, len(rows))
	for _, row := range rows {
		inputs = append(inputs, store.OpeningHourInput{
			DayOfWeek: row.Day,
			OpensAt:   row.Open,
			ClosesAt:  row.Close,
			Note:      row.Note,
		})
	}
	return inputs
}

func courseInputsFromForm(r *http.Request) []store.CourseInput {
	rows := courseRowsFromForm(r)
	inputs := make([]store.CourseInput, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Title) == "" {
			continue
		}
		inputs = append(inputs, store.CourseInput{
			DayOfWeek:   row.Day,
			Title:       row.Title,
			StartTime:   row.Start,
			EndTime:     row.End,
			Location:    row.Location,
			Instructor:  row.Instructor,
			Level:       row.Level,
			Description: row.Description,
		})
	}
	return inputs
}

func blankCourseRows(count int) []courseRow {
	rows := make([]courseRow, 0, count)
	for i := 0; i < count; i++ {
		rows = append(rows, courseRow{
			Day:      1,
			DayLabel: weekdayLabel(1),
		})
	}
	return rows
}

func trimTrailingEmptyCourseRows(rows []courseRow) []courseRow {
	cut := len(rows)
	for cut > 0 {
		row := rows[cut-1]
		if !isCourseRowEmpty(row) {
			break
		}
		cut--
	}
	return rows[:cut]
}

func isCourseRowEmpty(row courseRow) bool {
	return strings.TrimSpace(row.Title) == "" &&
		strings.TrimSpace(row.Start) == "" &&
		strings.TrimSpace(row.End) == "" &&
		strings.TrimSpace(row.Location) == "" &&
		strings.TrimSpace(row.Instructor) == "" &&
		strings.TrimSpace(row.Level) == "" &&
		strings.TrimSpace(row.Description) == ""
}

func weekdayLabel(day int) string {
	switch day {
	case 1:
		return "Montag"
	case 2:
		return "Dienstag"
	case 3:
		return "Mittwoch"
	case 4:
		return "Donnerstag"
	case 5:
		return "Freitag"
	case 6:
		return "Samstag"
	case 7:
		return "Sonntag"
	default:
		return ""
	}
}

func parseDay(value string, fallback int) int {
	day, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || day < 1 || day > 7 {
		return fallback
	}
	return day
}

func valueAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return values[index]
}

func maxLen(groups ...[]string) int {
	max := 0
	for _, group := range groups {
		if len(group) > max {
			max = len(group)
		}
	}
	return max
}
