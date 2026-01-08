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

const minPasswordLength = 8

type User struct {
	ID           string    `json:"id" gorm:"primaryKey;size:32"`
	Email        string    `json:"email" gorm:"uniqueIndex;size:320;not null"`
	PasswordHash string    `json:"password_hash" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Club struct {
	ID          string    `json:"id" gorm:"primaryKey;size:32"`
	OwnerID     string    `json:"owner_id" gorm:"uniqueIndex;size:32;not null"`
	Name        string    `json:"name" gorm:"not null"`
	Description string    `json:"description"`
	Slug        string    `json:"slug" gorm:"uniqueIndex;size:160;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Store struct {
	db             *gorm.DB
	policyMu       sync.RWMutex
	passwordPolicy PasswordPolicy
}

type PasswordPolicy struct {
	MinLength int
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

	if err := db.AutoMigrate(&User{}, &Club{}); err != nil {
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
	if err := s.db.Where("owner_id = ?", ownerID).First(&club).Error; err != nil {
		return Club{}, false
	}
	return club, true
}

func (s *Store) UpsertClub(ownerID, name, description string) (Club, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return Club{}, ErrNameRequired
	}

	cleanDescription := strings.TrimSpace(description)
	slugBase := slugify(cleanName)
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
			existing.Name = cleanName
			existing.Description = cleanDescription
			existing.Slug = uniqueSlug
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
			Name:        cleanName,
			Description: cleanDescription,
			Slug:        uniqueSlug,
			CreatedAt:   now,
			UpdatedAt:   now,
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

func (s *Store) AllClubs() []Club {
	var clubs []Club
	if err := s.db.Order("name asc").Order("slug asc").Find(&clubs).Error; err != nil {
		return []Club{}
	}
	return clubs
}

func (s *Store) minPasswordLength() int {
	s.policyMu.RLock()
	defer s.policyMu.RUnlock()
	if s.passwordPolicy.MinLength <= 0 {
		return minPasswordLength
	}
	return s.passwordPolicy.MinLength
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
