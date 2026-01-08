package site

import (
	"path"
	"path/filepath"

	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/ssgo/builder"
	"github.com/janmarkuslanger/ssgo/page"
	"github.com/janmarkuslanger/ssgo/rendering"
	"github.com/janmarkuslanger/ssgo/task"
	"github.com/janmarkuslanger/ssgo/taskutil"
	"github.com/janmarkuslanger/ssgo/writer"
)

type BuildOptions struct {
	OutputDir   string
	TemplateDir string
	AssetDir    string
}

func Build(clubs []store.Club, opts BuildOptions) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "public"
	}
	if opts.TemplateDir == "" {
		opts.TemplateDir = filepath.Join("templates", "site")
	}
	if opts.AssetDir == "" {
		opts.AssetDir = filepath.Join("static", "site")
	}

	clubBySlug := make(map[string]store.Club, len(clubs))
	paths := make([]string, 0, len(clubs))
	for _, club := range clubs {
		clubBySlug[club.Slug] = club
		paths = append(paths, path.Join("clubs", club.Slug, "index"))
	}

	generator := page.Generator{
		Config: page.Config{
			Template: filepath.Join(opts.TemplateDir, "club.html"),
			Pattern:  "clubs/:slug/index",
			GetPaths: func() []string {
				return paths
			},
			GetData: func(payload page.PagePayload) map[string]any {
				slug := payload.Params["slug"]
				club, ok := clubBySlug[slug]
				if !ok {
					return map[string]any{
						"Name":        "Club",
						"Description": "",
						"Slug":        slug,
					}
				}
				return map[string]any{
					"Name":        club.Name,
					"Description": club.Description,
					"Slug":        club.Slug,
				}
			},
			Renderer: rendering.HTMLRenderer{
				Layout: []string{filepath.Join(opts.TemplateDir, "layout.html")},
			},
		},
	}

	copyTask := taskutil.NewCopyTask(opts.AssetDir, "assets", nil)

	b := builder.Builder{
		OutputDir:  opts.OutputDir,
		Writer:     writer.NewFileWriter(),
		Generators: []page.Generator{generator},
		BeforeTasks: []task.Task{
			copyTask,
		},
	}

	return b.Build()
}
