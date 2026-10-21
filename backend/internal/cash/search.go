package cash

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	_ "time/tzdata" // Keep business dates available in minimal runtime images.

	"credit-union/backend/internal/platform/fault"
)

const (
	searchPageSize     = 50
	maxCursorLength    = 1024
	maxReferenceLength = 100
	cursorVersion      = 1
	businessTimeZone   = "Asia/Jakarta"
)

// SearchInput contains optional date, account, and pagination query values.
type SearchInput struct {
	Period, StartDate, EndDate, AccountNumber, Cursor string
}

// SearchPage is a bounded page of matching staff-facing transactions.
type SearchPage struct {
	Items      []ListedTransaction `json:"items"`
	NextCursor string              `json:"next_cursor"`
	StartDate  string              `json:"start_date"`
	EndDate    string              `json:"end_date"`
	TimeZone   string              `json:"time_zone"`
}

type searchCursor struct {
	Version       int       `json:"v"`
	CreatedAt     time.Time `json:"at"`
	Reference     string    `json:"ref"`
	AccountNumber string    `json:"account"`
	StartDate     string    `json:"start"`
	EndDate       string    `json:"end"`
}

type searchFilter struct {
	Start, End    time.Time // Exclusive end, both local midnights.
	AccountNumber string
	Before        *searchCursor
}

func invalidSearch() error {
	return fault.New(fault.Invalid, "Invalid date filter. Check period and start/end dates")
}
func invalidCursor() error {
	return fault.New(fault.Invalid, "Invalid pagination cursor. Apply the filters again")
}

func resolveSearch(input SearchInput, now time.Time) (searchFilter, error) {
	f := searchFilter{AccountNumber: strings.TrimSpace(input.AccountNumber)}
	if f.AccountNumber != "" && !transactionAccountPattern.MatchString(f.AccountNumber) {
		return f, fault.New(fault.Invalid, "Invalid account number. Enter exactly 10 digits")
	}
	loc, err := time.LoadLocation(businessTimeZone)
	if err != nil {
		return f, fmt.Errorf("load business timezone: %w", err)
	}
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	f.Start, f.End = today, today.AddDate(0, 0, 1)
	if input.Period != "custom" && (input.StartDate != "" || input.EndDate != "") {
		return f, invalidSearch()
	}
	switch input.Period {
	case "", "today":
	case "last_7_days":
		f.Start = today.AddDate(0, 0, -6)
	case "last_30_days":
		f.Start = today.AddDate(0, 0, -29)
	case "custom":
		start, e1 := time.ParseInLocation(time.DateOnly, input.StartDate, loc)
		end, e2 := time.ParseInLocation(time.DateOnly, input.EndDate, loc)
		if e1 != nil || e2 != nil || start.Year() < 1 || end.Year() < 1 || start.After(end) {
			return f, invalidSearch()
		}
		f.Start, f.End = start, end.AddDate(0, 0, 1)
	default:
		return f, invalidSearch()
	}
	if input.Cursor != "" {
		c, err := decodeSearchCursor(input.Cursor, f)
		if err != nil {
			return f, err
		}
		f.Before = &c
	}
	return f, nil
}

func decodeSearchCursor(encoded string, f searchFilter) (searchCursor, error) {
	var c searchCursor
	if len(encoded) > maxCursorLength {
		return c, invalidCursor()
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return c, invalidCursor()
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return c, invalidCursor()
	}
	if c.Version != cursorVersion || c.Reference == "" || len(c.Reference) > maxReferenceLength ||
		c.CreatedAt.Before(f.Start) || !c.CreatedAt.Before(f.End) ||
		c.AccountNumber != f.AccountNumber || c.StartDate != f.Start.Format(time.DateOnly) ||
		c.EndDate != f.End.AddDate(0, 0, -1).Format(time.DateOnly) {
		return c, invalidCursor()
	}
	return c, nil
}

// Search resolves business dates and returns a stable forward page without changing financial data.
func (s *Service) Search(ctx context.Context, input SearchInput) (SearchPage, error) {
	f, err := resolveSearch(input, time.Now())
	if err != nil {
		return SearchPage{}, err
	}
	rows, err := s.store.Search(ctx, f)
	if err != nil {
		return SearchPage{}, fmt.Errorf("search cash transactions: %w", err)
	}
	page := SearchPage{Items: rows, StartDate: f.Start.Format(time.DateOnly),
		EndDate: f.End.AddDate(0, 0, -1).Format(time.DateOnly), TimeZone: businessTimeZone}
	if len(rows) > searchPageSize {
		page.Items = rows[:searchPageSize]
		last := page.Items[len(page.Items)-1]
		data, err := json.Marshal(
			searchCursor{Version: cursorVersion, CreatedAt: last.CreatedAt, Reference: last.Reference,
				AccountNumber: f.AccountNumber, StartDate: page.StartDate, EndDate: page.EndDate},
		)
		if err != nil {
			return SearchPage{}, fmt.Errorf("encode cash page cursor: %w", err)
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
	}
	return page, nil
}
