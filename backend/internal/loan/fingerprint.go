package loan

// Legacy projections preserve persisted request hashes across DTO refactors.

type legacyInput struct {
	MemberID    int64  `json:"member_id"`
	Amount      int64  `json:"amount"`
	Tenor       int    `json:"tenor_months"`
	Purpose     string `json:"purpose"`
	Risk        string `json:"risk"`
	Responsible int    `json:"responsible_branch_id"`
}

func fingerprintInput(input Input) legacyInput {
	return legacyInput(input)
}

type legacyDecision struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func fingerprintDecision(input Decision) legacyDecision {
	return legacyDecision(input)
}
