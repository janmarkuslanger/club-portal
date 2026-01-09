package site

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

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

type openingHourView struct {
	Day   string
	Open  string
	Close string
	Note  string
}

type courseView struct {
	Title       string
	Start       string
	End         string
	Location    string
	Instructor  string
	Level       string
	Description string
}

type scheduleSlotView struct {
	Time    string
	Courses []courseView
}

type scheduleDayView struct {
	Day   string
	Slots []scheduleSlotView
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

	emptyOpening, _ := buildOpeningHours(nil)

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
						"Name":            "Club",
						"Description":     "",
						"Slug":            slug,
						"OpeningHours":    emptyOpening,
						"HasOpeningHours": false,
						"HasSchedule":     false,
						"HasContact":      false,
						"HasAddress":      false,
					}
				}

				openingHours, hasOpeningHours := buildOpeningHours(club.OpeningHours)
				schedule, hasSchedule := buildSchedule(club.Courses)
				hasContact := club.ContactName != "" || club.ContactRole != "" || club.ContactEmail != "" || club.ContactPhone != "" || club.ContactWebsite != ""
				hasAddress := club.AddressLine1 != "" || club.AddressLine2 != "" || club.AddressPostal != "" || club.AddressCity != "" || club.AddressCountry != ""

				return map[string]any{
					"Name":            club.Name,
					"Description":     club.Description,
					"Slug":            club.Slug,
					"ContactName":     club.ContactName,
					"ContactRole":     club.ContactRole,
					"ContactEmail":    club.ContactEmail,
					"ContactPhone":    club.ContactPhone,
					"ContactWebsite":  club.ContactWebsite,
					"AddressLine1":    club.AddressLine1,
					"AddressLine2":    club.AddressLine2,
					"AddressPostal":   club.AddressPostal,
					"AddressCity":     club.AddressCity,
					"AddressCountry":  club.AddressCountry,
					"OpeningHours":    openingHours,
					"HasOpeningHours": hasOpeningHours,
					"Schedule":        schedule,
					"HasSchedule":     hasSchedule,
					"HasContact":      hasContact,
					"HasAddress":      hasAddress,
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

func buildOpeningHours(hours []store.OpeningHour) ([]openingHourView, bool) {
	byDay := make(map[int]store.OpeningHour, len(hours))
	for _, hour := range hours {
		if hour.DayOfWeek < 1 || hour.DayOfWeek > 7 {
			continue
		}
		if _, exists := byDay[hour.DayOfWeek]; !exists {
			byDay[hour.DayOfWeek] = hour
		}
	}

	result := make([]openingHourView, 0, 7)
	hasAny := false
	for day := 1; day <= 7; day++ {
		hour := byDay[day]
		open := strings.TrimSpace(hour.OpensAt)
		close := strings.TrimSpace(hour.ClosesAt)
		note := strings.TrimSpace(hour.Note)
		if open != "" || close != "" || note != "" {
			hasAny = true
		}
		result = append(result, openingHourView{
			Day:   weekdayLabel(day),
			Open:  open,
			Close: close,
			Note:  note,
		})
	}

	return result, hasAny
}

func buildSchedule(courses []store.Course) ([]scheduleDayView, bool) {
	if len(courses) == 0 {
		return nil, false
	}

	sort.Slice(courses, func(i, j int) bool {
		if courses[i].DayOfWeek != courses[j].DayOfWeek {
			return courses[i].DayOfWeek < courses[j].DayOfWeek
		}
		startI := timeKey(courses[i].StartTime)
		startJ := timeKey(courses[j].StartTime)
		if startI != startJ {
			return startI < startJ
		}
		endI := timeKey(courses[i].EndTime)
		endJ := timeKey(courses[j].EndTime)
		if endI != endJ {
			return endI < endJ
		}
		return courses[i].Title < courses[j].Title
	})

	schedule := make([]scheduleDayView, 0)
	var currentDay *scheduleDayView
	var currentSlot *scheduleSlotView
	var currentDayValue int
	var currentSlotKey string

	for _, course := range courses {
		if course.DayOfWeek < 1 || course.DayOfWeek > 7 {
			continue
		}
		slotKey := timeKey(course.StartTime) + "|" + timeKey(course.EndTime)
		if currentDay == nil || currentDayValue != course.DayOfWeek {
			schedule = append(schedule, scheduleDayView{
				Day: weekdayLabel(course.DayOfWeek),
			})
			currentDay = &schedule[len(schedule)-1]
			currentSlot = nil
			currentDayValue = course.DayOfWeek
			currentSlotKey = ""
		}

		if currentSlot == nil || currentSlotKey != slotKey {
			currentDay.Slots = append(currentDay.Slots, scheduleSlotView{
				Time: formatTimeRange(course.StartTime, course.EndTime),
			})
			currentSlot = &currentDay.Slots[len(currentDay.Slots)-1]
			currentSlotKey = slotKey
		}

		currentSlot.Courses = append(currentSlot.Courses, courseView{
			Title:       course.Title,
			Start:       course.StartTime,
			End:         course.EndTime,
			Location:    course.Location,
			Instructor:  course.Instructor,
			Level:       course.Level,
			Description: course.Description,
		})
	}

	if len(schedule) == 0 {
		return nil, false
	}
	return schedule, true
}

func weekdayLabel(day int) string {
	switch day {
	case 1:
		return "Montag"
	case 2:
		return "Dienstag"
	case 3:
		return "Mittwoch"
	case 4:
		return "Donnerstag"
	case 5:
		return "Freitag"
	case 6:
		return "Samstag"
	case 7:
		return "Sonntag"
	default:
		return ""
	}
}

func timeKey(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == 4 && strings.Contains(value, ":") {
		return "0" + value
	}
	return value
}

func formatTimeRange(start, end string) string {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start != "" && end != "" {
		return start + " - " + end
	}
	if start != "" {
		return start
	}
	if end != "" {
		return end
	}
	return "nach Vereinbarung"
}
