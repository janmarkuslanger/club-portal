package main

import (
	"html/template"
	"path/filepath"
)

type templates struct {
	login     *template.Template
	register  *template.Template
	dashboard *template.Template
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
