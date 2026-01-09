package main

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
	ContactName     string
	ContactRole     string
	ContactEmail    string
	ContactPhone    string
	ContactWebsite  string
	AddressLine1    string
	AddressLine2    string
	AddressPostal   string
	AddressCity     string
	AddressCountry  string
	OpeningHours    []openingHourRow
	Courses         []courseRow
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
