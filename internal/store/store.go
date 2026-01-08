package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNameRequired       = errors.New("club name is required")
	ErrPasswordTooShort   = errors.New("password too short")
)

const minPasswordLength = 8

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type Club struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Slug        string    `json:"slug"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Store struct {
	mu             sync.RWMutex
	path           string
	users          map[string]User
	clubs          map[string]Club
	emailToUserID  map[string]string
	ownerToClubID  map[string]string
	slugToClubID   map[string]string
	passwordPolicy PasswordPolicy
}

type PasswordPolicy struct {
	MinLength int
}

type persistedData struct {
	Users []User `json:"users"`
	Clubs []Club `json:"clubs"`
}

func NewStore(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store path is required")
	}

	s := &Store{
		path:          path,
		users:         make(map[string]User),
		clubs:         make(map[string]Club),
		emailToUserID: make(map[string]string),
		ownerToClubID: make(map[string]string),
		slugToClubID:  make(map[string]string),
		passwordPolicy: PasswordPolicy{
			MinLength: minPasswordLength,
		},
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) SetPasswordPolicy(policy PasswordPolicy) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if policy.MinLength <= 0 {
		policy.MinLength = minPasswordLength
	}
	s.passwordPolicy = policy
}

func (s *Store) CreateUser(email, password string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanEmail := normalizeEmail(email)
	if cleanEmail == "" {
		return User{}, errors.New("email is required")
	}
	if len(password) < s.passwordPolicy.MinLength {
		return User{}, ErrPasswordTooShort
	}
	if _, exists := s.emailToUserID[cleanEmail]; exists {
		return User{}, ErrEmailExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	now := time.Now().UTC()
	user := User{
		ID:           newID(),
		Email:        cleanEmail,
		PasswordHash: string(hash),
		CreatedAt:    now,
	}

	s.users[user.ID] = user
	s.emailToUserID[cleanEmail] = user.ID

	if err := s.saveLocked(); err != nil {
		return User{}, err
	}

	return user, nil
}

func (s *Store) Authenticate(email, password string) (User, error) {
	s.mu.RLock()
	userID, ok := s.emailToUserID[normalizeEmail(email)]
	if !ok {
		s.mu.RUnlock()
		return User{}, ErrInvalidCredentials
	}
	user := s.users[userID]
	s.mu.RUnlock()

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Store) GetUser(id string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	return user, ok
}

func (s *Store) GetClubByOwner(ownerID string) (Club, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clubID, ok := s.ownerToClubID[ownerID]
	if !ok {
		return Club{}, false
	}
	club, ok := s.clubs[clubID]
	return club, ok
}

func (s *Store) UpsertClub(ownerID, name, description string) (Club, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return Club{}, ErrNameRequired
	}

	cleanDescription := strings.TrimSpace(description)
	newSlug := slugify(cleanName)
	if newSlug == "" {
		newSlug = "club"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	clubID, exists := s.ownerToClubID[ownerID]
	if exists {
		club := s.clubs[clubID]
		uniqueSlug := s.uniqueSlugLocked(club.ID, newSlug)
		if club.Slug != uniqueSlug {
			delete(s.slugToClubID, club.Slug)
			s.slugToClubID[uniqueSlug] = club.ID
		}

		club.Name = cleanName
		club.Description = cleanDescription
		club.Slug = uniqueSlug
		club.UpdatedAt = now
		s.clubs[club.ID] = club

		if err := s.saveLocked(); err != nil {
			return Club{}, err
		}
		return club, nil
	}

	club := Club{
		ID:          newID(),
		OwnerID:     ownerID,
		Name:        cleanName,
		Description: cleanDescription,
		Slug:        s.uniqueSlugLocked("", newSlug),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.clubs[club.ID] = club
	s.ownerToClubID[ownerID] = club.ID
	s.slugToClubID[club.Slug] = club.ID

	if err := s.saveLocked(); err != nil {
		return Club{}, err
	}
	return club, nil
}

func (s *Store) AllClubs() []Club {
	s.mu.RLock()
	clubs := make([]Club, 0, len(s.clubs))
	for _, club := range s.clubs {
		clubs = append(clubs, club)
	}
	s.mu.RUnlock()

	sort.Slice(clubs, func(i, j int) bool {
		if clubs[i].Name == clubs[j].Name {
			return clubs[i].Slug < clubs[j].Slug
		}
		return clubs[i].Name < clubs[j].Name
	})

	return clubs
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var payload persistedData
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}

	for _, user := range payload.Users {
		s.users[user.ID] = user
		s.emailToUserID[normalizeEmail(user.Email)] = user.ID
	}

	for _, club := range payload.Clubs {
		s.clubs[club.ID] = club
		s.ownerToClubID[club.OwnerID] = club.ID
		s.slugToClubID[club.Slug] = club.ID
	}

	return nil
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	payload := persistedData{
		Users: make([]User, 0, len(s.users)),
		Clubs: make([]Club, 0, len(s.clubs)),
	}
	for _, user := range s.users {
		payload.Users = append(payload.Users, user)
	}
	for _, club := range s.clubs {
		payload.Clubs = append(payload.Clubs, club)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

func (s *Store) uniqueSlugLocked(currentClubID, desired string) string {
	base := desired
	if base == "" {
		base = "club"
	}

	slug := base
	for i := 2; ; i++ {
		existingID, exists := s.slugToClubID[slug]
		if !exists || existingID == currentClubID {
			return slug
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
