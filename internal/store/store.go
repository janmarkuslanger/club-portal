package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNameRequired       = errors.New("club name is required")
	ErrPasswordTooShort   = errors.New("password too short")
)

const (
	minPasswordLength  = 8
	buildTaskKey       = "site_build"
	buildStatusIdle    = "idle"
	buildStatusPending = "pending"
	buildStatusRunning = "running"
)

type User struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	Email        string    `json:"email" gorm:"uniqueIndex;size:320;not null"`
	PasswordHash string    `json:"password_hash" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Club struct {
	ID          string `json:"id" gorm:"primaryKey;size:32"`
	OwnerID     string `json:"owner_id" gorm:"uniqueIndex;size:32;not null"`
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description"`
	Categories  string `json:"categories" gorm:"size:400"`
	Slug        string `json:"slug" gorm:"uniqueIndex;size:160;not null"`

	ContactName    string `json:"contact_name" gorm:"size:120"`
	ContactRole    string `json:"contact_role" gorm:"size:120"`
	ContactEmail   string `json:"contact_email" gorm:"size:320"`
	ContactPhone   string `json:"contact_phone" gorm:"size:50"`
	ContactWebsite string `json:"contact_website" gorm:"size:200"`

	AddressLine1   string `json:"address_line_1" gorm:"size:200"`
	AddressLine2   string `json:"address_line_2" gorm:"size:200"`
	AddressPostal  string `json:"address_postal" gorm:"size:20"`
	AddressCity    string `json:"address_city" gorm:"size:120"`
	AddressCountry string `json:"address_country" gorm:"size:120"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	OpeningHours []OpeningHour `json:"opening_hours" gorm:"foreignKey:ClubID;references:ID;constraint:OnDelete:CASCADE"`
	Courses      []Course      `json:"courses" gorm:"foreignKey:ClubID;references:ID;constraint:OnDelete:CASCADE"`
}

type OpeningHour struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	ClubID    string `json:"club_id" gorm:"index;size:32;not null"`
	DayOfWeek int    `json:"day_of_week" gorm:"not null"`
	OpensAt   string `json:"opens_at" gorm:"size:5"`
	ClosesAt  string `json:"closes_at" gorm:"size:5"`
	Note      string `json:"note" gorm:"size:200"`
}

type Course struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	ClubID      string `json:"club_id" gorm:"index;size:32;not null"`
	DayOfWeek   int    `json:"day_of_week" gorm:"not null"`
	Title       string `json:"title" gorm:"not null"`
	StartTime   string `json:"start_time" gorm:"size:5"`
	EndTime     string `json:"end_time" gorm:"size:5"`
	Location    string `json:"location" gorm:"size:120"`
	Instructor  string `json:"instructor" gorm:"size:120"`
	Level       string `json:"level" gorm:"size:120"`
	Description string `json:"description" gorm:"size:400"`
}

type BuildTask struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Key         string    `json:"key" gorm:"uniqueIndex;size:40;not null"`
	Status      string    `json:"status" gorm:"size:20;not null"`
	NextRunAt   time.Time `json:"next_run_at"`
	LastEventAt time.Time `json:"last_event_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Store struct {
	db             *gorm.DB
	policyMu       sync.RWMutex
	passwordPolicy PasswordPolicy
}

type PasswordPolicy struct {
	MinLength int
}

type ClubUpdate struct {
	Name        string
	Description string
	Categories  string

	ContactName    string
	ContactRole    string
	ContactEmail   string
	ContactPhone   string
	ContactWebsite string

	AddressLine1   string
	AddressLine2   string
	AddressPostal  string
	AddressCity    string
	AddressCountry string
}

type OpeningHourInput struct {
	DayOfWeek int
	OpensAt   string
	ClosesAt  string
	Note      string
}

type CourseInput struct {
	DayOfWeek   int
	Title       string
	StartTime   string
	EndTime     string
	Location    string
	Instructor  string
	Level       string
	Description string
}

type ExampleSeed struct {
	Email        string
	Password     string
	Club         Club
	OpeningHours []OpeningHourInput
	Courses      []CourseInput
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store path is required")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&User{}, &Club{}, &OpeningHour{}, &Course{}, &BuildTask{}); err != nil {
		return nil, err
	}

	return &Store{
		db: db,
		passwordPolicy: PasswordPolicy{
			MinLength: minPasswordLength,
		},
	}, nil
}

func (s *Store) SetPasswordPolicy(policy PasswordPolicy) {
	s.policyMu.Lock()
	defer s.policyMu.Unlock()

	if policy.MinLength <= 0 {
		policy.MinLength = minPasswordLength
	}
	s.passwordPolicy = policy
}

func (s *Store) CreateUser(email, password string) (User, error) {
	cleanEmail := normalizeEmail(email)
	if cleanEmail == "" {
		return User{}, errors.New("email is required")
	}
	if len(password) < s.minPasswordLength() {
		return User{}, ErrPasswordTooShort
	}

	var existing User
	err := s.db.Select("id").Where("email = ?", cleanEmail).First(&existing).Error
	if err == nil {
		return User{}, ErrEmailExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	user := User{
		ID:           newID(),
		Email:        cleanEmail,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.db.Create(&user).Error; err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *Store) Authenticate(email, password string) (User, error) {
	var user User
	err := s.db.Where("email = ?", normalizeEmail(email)).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Store) GetUser(id string) (User, bool) {
	var user User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return User{}, false
	}
	return user, true
}

func (s *Store) GetClubByOwner(ownerID string) (Club, bool) {
	var club Club
	if err := s.db.Preload("OpeningHours", orderOpeningHours).
		Preload("Courses", orderCourses).
		Where("owner_id = ?", ownerID).First(&club).Error; err != nil {
		return Club{}, false
	}
	return club, true
}

func (s *Store) UpsertClub(ownerID string, update ClubUpdate) (Club, error) {
	clean := sanitizeClubUpdate(update)
	if clean.Name == "" {
		return Club{}, ErrNameRequired
	}

	slugBase := slugify(clean.Name)
	if slugBase == "" {
		slugBase = "club"
	}

	var result Club
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var existing Club
		err := tx.Where("owner_id = ?", ownerID).First(&existing).Error
		hasExisting := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		currentID := ""
		if hasExisting {
			currentID = existing.ID
		}

		uniqueSlug, err := uniqueSlug(tx, currentID, slugBase)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		if hasExisting {
			existing.Name = clean.Name
			existing.Description = clean.Description
			existing.Categories = clean.Categories
			existing.Slug = uniqueSlug

			existing.ContactName = clean.ContactName
			existing.ContactRole = clean.ContactRole
			existing.ContactEmail = clean.ContactEmail
			existing.ContactPhone = clean.ContactPhone
			existing.ContactWebsite = clean.ContactWebsite
			existing.AddressLine1 = clean.AddressLine1
			existing.AddressLine2 = clean.AddressLine2
			existing.AddressPostal = clean.AddressPostal
			existing.AddressCity = clean.AddressCity
			existing.AddressCountry = clean.AddressCountry

			existing.UpdatedAt = now
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			result = existing
			return nil
		}

		club := Club{
			ID:          newID(),
			OwnerID:     ownerID,
			Name:        clean.Name,
			Description: clean.Description,
			Categories:  clean.Categories,
			Slug:        uniqueSlug,

			ContactName:    clean.ContactName,
			ContactRole:    clean.ContactRole,
			ContactEmail:   clean.ContactEmail,
			ContactPhone:   clean.ContactPhone,
			ContactWebsite: clean.ContactWebsite,
			AddressLine1:   clean.AddressLine1,
			AddressLine2:   clean.AddressLine2,
			AddressPostal:  clean.AddressPostal,
			AddressCity:    clean.AddressCity,
			AddressCountry: clean.AddressCountry,

			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := tx.Create(&club).Error; err != nil {
			return err
		}
		result = club
		return nil
	})
	if err != nil {
		return Club{}, err
	}

	return result, nil
}

func (s *Store) ReplaceOpeningHours(clubID string, hours []OpeningHourInput) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("club_id = ?", clubID).Delete(&OpeningHour{}).Error; err != nil {
			return err
		}

		items := make([]OpeningHour, 0, len(hours))
		for _, hour := range hours {
			if hour.DayOfWeek < 1 || hour.DayOfWeek > 7 {
				continue
			}
			opens := strings.TrimSpace(hour.OpensAt)
			closes := strings.TrimSpace(hour.ClosesAt)
			note := strings.TrimSpace(hour.Note)
			if opens == "" && closes == "" && note == "" {
				continue
			}
			items = append(items, OpeningHour{
				ClubID:    clubID,
				DayOfWeek: hour.DayOfWeek,
				OpensAt:   opens,
				ClosesAt:  closes,
				Note:      note,
			})
		}

		if len(items) == 0 {
			return nil
		}

		return tx.Create(&items).Error
	})
}

func (s *Store) ReplaceCourses(clubID string, courses []CourseInput) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("club_id = ?", clubID).Delete(&Course{}).Error; err != nil {
			return err
		}

		items := make([]Course, 0, len(courses))
		for _, course := range courses {
			title := strings.TrimSpace(course.Title)
			if title == "" {
				continue
			}
			day := course.DayOfWeek
			if day < 1 || day > 7 {
				continue
			}
			items = append(items, Course{
				ClubID:      clubID,
				DayOfWeek:   day,
				Title:       title,
				StartTime:   strings.TrimSpace(course.StartTime),
				EndTime:     strings.TrimSpace(course.EndTime),
				Location:    strings.TrimSpace(course.Location),
				Instructor:  strings.TrimSpace(course.Instructor),
				Level:       strings.TrimSpace(course.Level),
				Description: strings.TrimSpace(course.Description),
			})
		}

		if len(items) == 0 {
			return nil
		}

		return tx.Create(&items).Error
	})
}

func (s *Store) AllClubs() []Club {
	var clubs []Club
	if err := s.db.Preload("OpeningHours", orderOpeningHours).
		Preload("Courses", orderCourses).
		Order("name asc").Order("slug asc").Find(&clubs).Error; err != nil {
		return []Club{}
	}
	return clubs
}

func (s *Store) EnsureExampleClub() (ExampleSeed, bool, error) {
	var count int64
	if err := s.db.Table("clubs").
		Joins("JOIN users ON users.id = clubs.owner_id").
		Count(&count).Error; err != nil {
		return ExampleSeed{}, false, err
	}
	if count > 0 {
		return ExampleSeed{}, false, nil
	}

	email := "demo@club-portal.test"
	password := "demo1234"

	user, err := s.CreateUser(email, password)
	if err != nil {
		return ExampleSeed{}, false, err
	}

	update := ClubUpdate{
		Name:        "SV Morgenrot 1922",
		Description: "Wir sind ein offener Mehrspartenverein mit Fokus auf Gemeinschaft, Gesundheit und sportliche Vielfalt. Neue Mitglieder sind jederzeit willkommen.",
		Categories:  "Fitness, Yoga, Gesundheitssport",

		ContactName:    "Lena Berger",
		ContactRole:    "Vereinsleitung",
		ContactEmail:   "kontakt@sv-morgenrot.de",
		ContactPhone:   "+49 30 1234567",
		ContactWebsite: "https://sv-morgenrot.de",

		AddressLine1:   "Sportpark Nord",
		AddressLine2:   "Hallenweg 12",
		AddressPostal:  "10115",
		AddressCity:    "Berlin",
		AddressCountry: "Deutschland",
	}

	club, err := s.UpsertClub(user.ID, update)
	if err != nil {
		return ExampleSeed{}, false, err
	}

	openingHours := []OpeningHourInput{
		{DayOfWeek: 1, OpensAt: "09:00", ClosesAt: "12:00"},
		{DayOfWeek: 2, OpensAt: "16:00", ClosesAt: "20:00"},
		{DayOfWeek: 3, OpensAt: "09:00", ClosesAt: "12:00"},
		{DayOfWeek: 4, OpensAt: "16:00", ClosesAt: "20:00"},
		{DayOfWeek: 5, OpensAt: "14:00", ClosesAt: "18:00"},
		{DayOfWeek: 6, OpensAt: "10:00", ClosesAt: "13:00"},
		{DayOfWeek: 7, Note: "geschlossen"},
	}
	if err := s.ReplaceOpeningHours(club.ID, openingHours); err != nil {
		return ExampleSeed{}, false, err
	}

	courses := []CourseInput{
		{DayOfWeek: 2, Title: "Functional Training", StartTime: "18:00", EndTime: "19:30", Location: "Halle A", Instructor: "Mara Stein", Level: "Alle Level"},
		{DayOfWeek: 2, Title: "Yoga Flow", StartTime: "18:00", EndTime: "19:30", Location: "Studio 2", Instructor: "Jonas Weber", Level: "Einsteiger"},
		{DayOfWeek: 3, Title: "Kinderturnen", StartTime: "17:00", EndTime: "18:30", Location: "Halle B", Instructor: "Lea Schmitt", Level: "6-9 Jahre"},
		{DayOfWeek: 4, Title: "Badminton Freies Spiel", StartTime: "19:00", EndTime: "20:30", Location: "Halle C", Instructor: "Team"},
		{DayOfWeek: 6, Title: "Lauftreff", StartTime: "09:30", EndTime: "11:00", Location: "Parkrunde", Instructor: "Max Urban", Level: "Alle Level"},
	}
	if err := s.ReplaceCourses(club.ID, courses); err != nil {
		return ExampleSeed{}, false, err
	}

	club.OpeningHours = nil
	club.Courses = nil
	return ExampleSeed{
		Email:        email,
		Password:     password,
		Club:         club,
		OpeningHours: openingHours,
		Courses:      courses,
	}, true, nil
}

func (s *Store) EnqueueBuildTask(debounce time.Duration) error {
	now := time.Now().UTC()
	if debounce < 0 {
		debounce = 0
	}
	next := now.Add(debounce)

	return s.db.Transaction(func(tx *gorm.DB) error {
		var task BuildTask
		err := tx.Where("key = ?", buildTaskKey).First(&task).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			task = BuildTask{
				Key:         buildTaskKey,
				Status:      buildStatusPending,
				NextRunAt:   next,
				LastEventAt: now,
			}
			return tx.Create(&task).Error
		}
		if err != nil {
			return err
		}

		task.NextRunAt = next
		task.LastEventAt = now
		if task.Status != buildStatusRunning {
			task.Status = buildStatusPending
		}

		return tx.Save(&task).Error
	})
}

func (s *Store) ClaimBuildTask(now time.Time) (BuildTask, bool, error) {
	var task BuildTask
	err := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&BuildTask{}).
			Where("key = ? AND status = ? AND next_run_at <= ?", buildTaskKey, buildStatusPending, now).
			Updates(map[string]any{
				"status": buildStatusRunning,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("key = ?", buildTaskKey).First(&task).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return BuildTask{}, false, nil
	}
	if err != nil {
		return BuildTask{}, false, err
	}
	return task, true, nil
}

func (s *Store) CompleteBuildTask(taskID uint) error {
	now := time.Now().UTC()
	var task BuildTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return err
	}

	if task.NextRunAt.After(now) {
		task.Status = buildStatusPending
	} else {
		task.Status = buildStatusIdle
		task.NextRunAt = time.Time{}
	}

	return s.db.Save(&task).Error
}

func (s *Store) RescheduleBuildTask(taskID uint, delay time.Duration) error {
	if delay < 0 {
		delay = 0
	}
	next := time.Now().UTC().Add(delay)

	return s.db.Model(&BuildTask{}).
		Where("id = ?", taskID).
		Updates(map[string]any{
			"status":      buildStatusPending,
			"next_run_at": next,
		}).Error
}

func (s *Store) minPasswordLength() int {
	s.policyMu.RLock()
	defer s.policyMu.RUnlock()
	if s.passwordPolicy.MinLength <= 0 {
		return minPasswordLength
	}
	return s.passwordPolicy.MinLength
}

func orderOpeningHours(db *gorm.DB) *gorm.DB {
	return db.Order("day_of_week asc").Order("opens_at asc")
}

func orderCourses(db *gorm.DB) *gorm.DB {
	return db.Order("day_of_week asc").Order("start_time asc").Order("title asc")
}

func sanitizeClubUpdate(update ClubUpdate) ClubUpdate {
	update.Name = strings.TrimSpace(update.Name)
	update.Description = strings.TrimSpace(update.Description)
	update.Categories = normalizeCategories(update.Categories)
	update.ContactName = strings.TrimSpace(update.ContactName)
	update.ContactRole = strings.TrimSpace(update.ContactRole)
	update.ContactEmail = strings.TrimSpace(update.ContactEmail)
	update.ContactPhone = strings.TrimSpace(update.ContactPhone)
	update.ContactWebsite = strings.TrimSpace(update.ContactWebsite)
	update.AddressLine1 = strings.TrimSpace(update.AddressLine1)
	update.AddressLine2 = strings.TrimSpace(update.AddressLine2)
	update.AddressPostal = strings.TrimSpace(update.AddressPostal)
	update.AddressCity = strings.TrimSpace(update.AddressCity)
	update.AddressCountry = strings.TrimSpace(update.AddressCountry)
	return update
}

func NormalizeCategories(raw string) string {
	return normalizeCategories(raw)
}

func SplitCategories(raw string) []string {
	return splitCategories(raw)
}

func normalizeCategories(raw string) string {
	items := splitCategories(raw)
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, ", ")
}

func splitCategories(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '\n', '\r', '\t':
			return true
		default:
			return false
		}
	})
	seen := make(map[string]struct{})
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		item = strings.Join(strings.Fields(item), " ")
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, item)
	}
	return items
}

func uniqueSlug(tx *gorm.DB, currentClubID, desired string) (string, error) {
	base := desired
	if base == "" {
		base = "club"
	}

	slug := base
	for i := 2; ; i++ {
		query := tx.Model(&Club{}).Where("slug = ?", slug)
		if currentClubID != "" {
			query = query.Where("id <> ?", currentClubID)
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func slugify(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"ä", "ae",
		"ö", "oe",
		"ü", "ue",
		"ß", "ss",
	)
	input = replacer.Replace(input)

	var b strings.Builder
	b.Grow(len(input))
	lastDash := false
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	slug := b.String()
	slug = strings.Trim(slug, "-")
	return slug
}
