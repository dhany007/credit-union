package cash

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"credit-union/backend/internal/platform/fault"
)

var transactionAccountPattern = regexp.MustCompile(`^[1-9][0-9]{9}$`)

// List lists recent cooperative-wide cash postings with an optional exact account filter.
func (s *Service) List(ctx context.Context, accountNumber string) ([]ListedTransaction, error) {
	accountNumber = strings.TrimSpace(accountNumber)
	if accountNumber != "" && !transactionAccountPattern.MatchString(accountNumber) {
		return nil, fault.New(fault.Invalid, "Invalid account number. Enter exactly 10 digits")
	}
	rows, err := s.store.List(ctx, accountNumber)
	if err != nil {
		return nil, fmt.Errorf("list cash transactions: %w", err)
	}
	return rows, nil
}
