package cash

import "time"

// Input contains the cash command fields and their legacy JSON names.
type Input struct {
	MemberID int64  `json:"member_id"`
	Amount   int64  `json:"amount"`
	Kind     string `json:"kind"`
	Note     string `json:"note"`
}

// Result contains the public cash operation result.
type Result struct {
	Reference string `json:"reference"`
}

// Transaction is the staff-facing cash history projection.
type Transaction struct {
	BalanceBefore *int64    `json:"balance_before"`
	BalanceAfter  *int64    `json:"balance_after"`
	Reference     string    `json:"reference"`
	Kind          string    `json:"kind"`
	Amount        int64     `json:"amount"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"created_at"`
	Branch        string    `json:"branch"`
	Actor         string    `json:"actor"`
}

// Branch identifies a configured cooperative service location.
type Branch struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ListedTransaction adds member attribution to the cooperative-wide staff feed.
// Snapshot fields are shared with the member-specific history projection.
type ListedTransaction struct {
	Transaction
	MemberID      int64  `json:"member_id"`
	MemberNumber  string `json:"member_number"`
	MemberName    string `json:"member_name"`
	AccountNumber string `json:"account_number"`
}
