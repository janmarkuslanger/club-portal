package main

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
	AppName         string
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

type homeData struct {
	AppName    string
	Title      string
	ClubCount  int
	Cities     []string
	Clubs      []homeClub
}

type homeClub struct {
	Name       string
	Slug       string
	Description string
	Location   string
	City       string
	SearchText string
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
