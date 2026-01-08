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
}
