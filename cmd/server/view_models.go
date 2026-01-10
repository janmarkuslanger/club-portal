package main

import "html/template"

type loginData struct {
	AppName string
	Title   string
	Error   string
	Email   string
}

type registerData struct {
	AppName string
	Title   string
	Error   string
	Email   string
}

type dashboardData struct {
	AppName           string
	Title             string
	Error             string
	Info              string
	ClubName          string
	ClubDescription   string
	ClubCategories    string
	CategoryOptions   []categoryOption
	CategorySelection map[string]bool
	CategoryCustom    string
	ClubSlug          string
	PreviewPath       string
	ContactName       string
	ContactRole       string
	ContactEmail      string
	ContactPhone      string
	ContactWebsite    string
	AddressLine1      string
	AddressLine2      string
	AddressPostal     string
	AddressCity       string
	AddressCountry    string
	OpeningHours      []openingHourRow
	Courses           []courseRow
}

type homeData struct {
	AppName    string
	Title      string
	ClubCount  int
	Cities     []string
	Categories []homeCategory
	Clubs      []homeClub
}

type homeCategory struct {
	Value string
	Label string
	Icon  template.HTML
}

type homeClub struct {
	Name           string
	Slug           string
	Description    string
	Location       string
	City           string
	Categories     []string
	SearchText     string
	CategorySearch string
}

type openingHourRow struct {
	Day      int
	DayLabel string
	Open     string
	Close    string
	Note     string
}

type courseRow struct {
	Day         int
	DayLabel    string
	Title       string
	Start       string
	End         string
	Location    string
	Instructor  string
	Level       string
	Description string
}
