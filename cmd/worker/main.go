package main

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/janmarkuslanger/club-portal/internal/site"
	"github.com/janmarkuslanger/club-portal/internal/store"
)

const (
	defaultDataPath     = "data/store.db"
	defaultOutputDir    = "public"
	defaultTemplateDir  = "templates/site"
	defaultAssetDir     = "static/site"
	defaultPollInterval = 5 * time.Second
	defaultRetryDelay   = 5 * time.Minute
	defaultNightlyAt    = "03:00"
)

func main() {
	dataPath := envOrDefault("DATA_PATH", defaultDataPath)
	outputDir := envOrDefault("OUTPUT_DIR", defaultOutputDir)
	templateDir := envOrDefault("TEMPLATE_DIR", defaultTemplateDir)
	assetDir := envOrDefault("ASSET_DIR", defaultAssetDir)
	pollInterval := envDuration("BUILD_POLL_INTERVAL", defaultPollInterval)
	retryDelay := envDuration("BUILD_RETRY_DELAY", defaultRetryDelay)
	nightlyAt := envOrDefault("BUILD_NIGHTLY_AT", defaultNightlyAt)

	storeInstance, err := store.NewStore(dataPath)
	if err != nil {
		log.Fatal(err)
	}

	buildOptions := site.BuildOptions{
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	}

	nextNightly, err := nextNightlyRun(time.Now(), nightlyAt)
	if err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Printf("build worker started (nightly at %s)", nightlyAt)

	for {
		now := time.Now()
		if now.After(nextNightly) || now.Equal(nextNightly) {
			if err := storeInstance.EnqueueBuildTask(0); err != nil {
				log.Printf("nightly enqueue failed: %v", err)
			} else {
				log.Println("nightly build enqueued")
			}
			nextNightly, _ = nextNightlyRun(now.Add(time.Minute), nightlyAt)
		}

		if err := processBuildQueue(storeInstance, buildOptions, retryDelay); err != nil {
			log.Printf("build queue error: %v", err)
		}

		<-ticker.C
	}
}

func processBuildQueue(storeInstance *store.Store, options site.BuildOptions, retryDelay time.Duration) error {
	now := time.Now().UTC()
	task, ok, err := storeInstance.ClaimBuildTask(now)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	log.Printf("build task claimed (next run scheduled at %s)", task.NextRunAt.Format(time.RFC3339))
	clubs := storeInstance.AllClubs()
	if err := site.Build(clubs, options); err != nil {
		log.Printf("build failed: %v", err)
		return storeInstance.RescheduleBuildTask(task.ID, retryDelay)
	}

	if err := storeInstance.CompleteBuildTask(task.ID); err != nil {
		return err
	}

	log.Printf("build finished (%d clubs)", len(clubs))
	return nil
}

func nextNightlyRun(now time.Time, at string) (time.Time, error) {
	parts := strings.Split(at, ":")
	if len(parts) != 2 {
		return time.Time{}, errors.New("BUILD_NIGHTLY_AT must be HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return time.Time{}, errors.New("BUILD_NIGHTLY_AT hour invalid")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return time.Time{}, errors.New("BUILD_NIGHTLY_AT minute invalid")
	}

	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next, nil
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
