package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/janmarkuslanger/club-portal/internal/auth"
	"github.com/janmarkuslanger/club-portal/internal/site"
	"github.com/janmarkuslanger/club-portal/internal/store"
	"github.com/janmarkuslanger/graft/graft"
)

const (
	defaultDataPath    = "data/store.db"
	defaultOutputDir   = "public"
	defaultTemplateDir = "templates/site"
	defaultAssetDir    = "static/site"
)

func main() {
	dataPath := envOrDefault("DATA_PATH", defaultDataPath)
	outputDir := envOrDefault("OUTPUT_DIR", defaultOutputDir)
	templateDir := envOrDefault("TEMPLATE_DIR", defaultTemplateDir)
	assetDir := envOrDefault("ASSET_DIR", defaultAssetDir)

	storeInstance, err := store.NewStore(dataPath)
	if err != nil {
		log.Fatal(err)
	}

	sessions := auth.NewManager(envDuration("SESSION_TTL", 24*time.Hour))
	cookieSecure := envBool("COOKIE_SECURE", false)

	tmpls, err := loadTemplates(filepath.Join("templates", "admin"))
	if err != nil {
		log.Fatal(err)
	}

	buildOptions := site.BuildOptions{
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	}

	app := graft.New()
	app.UseModule(seedModule{
		Store: storeInstance,
	})
	app.UseModule(staticModule{
		AdminAssetsDir: filepath.Join("static", "admin"),
		SiteAssetsDir:  filepath.Join(outputDir, "assets"),
		ClubsDir:       filepath.Join(outputDir, "clubs"),
	})

	app.UseModule(publicModule(publicDeps{
		Sessions:  sessions,
		Templates: tmpls,
	}))
	app.UseModule(authModule(authDeps{
		Store:        storeInstance,
		Sessions:     sessions,
		Templates:    tmpls,
		CookieSecure: cookieSecure,
	}))
	app.UseModule(adminModule(adminDeps{
		Store:        storeInstance,
		Sessions:     sessions,
		Templates:    tmpls,
		BuildOptions: buildOptions,
		CookieSecure: cookieSecure,
	}))

	log.Println("club-portal server running on :8080")
	app.Run()
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
