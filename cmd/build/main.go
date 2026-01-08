package main

import (
	"log"
	"os"
	"strings"

	"github.com/janmarkuslanger/club-portal/internal/site"
	"github.com/janmarkuslanger/club-portal/internal/store"
)

const (
	defaultDataPath    = "data/store.json"
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

	clubs := storeInstance.AllClubs()
	if err := site.Build(clubs, site.BuildOptions{
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	}); err != nil {
		log.Fatal(err)
	}

	log.Printf("static site built: %d clubs -> %s", len(clubs), outputDir)
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
