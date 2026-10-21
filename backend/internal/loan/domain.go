package loan

import "time"

// Input contains the loan command fields and their legacy JSON names.
type Input struct {
	MemberID    int64  `json:"member_id"`
	Amount      int64  `json:"amount"`
	Tenor       int    `json:"tenor_months"`
	Purpose     string `json:"purpose"`
	Risk        string `json:"risk"`
	Responsible int    `json:"responsible_branch_id"`
}

// Decision contains a requested application outcome and verification reason.
type Decision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// Application is the staff-facing application projection, not an active loan.
type Application struct {
	ID                int64     `json:"id"`
	MemberID          int64     `json:"member_id"`
	MemberName        string    `json:"member_name"`
	Amount            int64     `json:"amount"`
	Tenor             int       `json:"tenor_months"`
	Purpose           string    `json:"purpose"`
	Risk              string    `json:"risk"`
	Status            string    `json:"status"`
	ReceivingBranch   string    `json:"receiving_branch"`
	ResponsibleID     int       `json:"responsible_branch_id"`
	ResponsibleBranch string    `json:"responsible_branch"`
	CreatedAt         time.Time `json:"created_at"`
	DecisionReason    string    `json:"decision_reason"`
	DecisionBy        string    `json:"decision_by"`
}

// LockedApplication contains decision facts read under a row lock.
type LockedApplication struct {
	Amount, Creator       int64
	Product, Risk, Status string
}
