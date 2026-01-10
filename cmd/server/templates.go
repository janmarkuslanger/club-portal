package main

import (
	"html/template"
	"path/filepath"
)

type templates struct {
	login     *template.Template
	register  *template.Template
	dashboard *template.Template
	home      *template.Template
}

func loadTemplates(dir string) (templates, error) {
	funcs := template.FuncMap{
		"eq": func(a, b any) bool { return a == b },
	}

	login, err := template.New("login.html").Funcs(funcs).ParseFiles(filepath.Join(dir, "login.html"))
	if err != nil {
		return templates{}, err
	}
	register, err := template.New("register.html").Funcs(funcs).ParseFiles(filepath.Join(dir, "register.html"))
	if err != nil {
		return templates{}, err
	}
	dashboard, err := template.New("dashboard.html").Funcs(funcs).ParseFiles(filepath.Join(dir, "dashboard.html"))
	if err != nil {
		return templates{}, err
	}
	home, err := template.New("home.html").Funcs(funcs).ParseFiles(filepath.Join("templates", "public", "home.html"))
	if err != nil {
		return templates{}, err
	}

	return templates{
		login:     login,
		register:  register,
		dashboard: dashboard,
		home:      home,
	}, nil
}
