package cash

// Legacy projections preserve persisted request hashes across DTO refactors.

type legacyInput struct {
	MemberID int64  `json:"member_id"`
	Amount   int64  `json:"amount"`
	Kind     string `json:"kind"`
	Note     string `json:"note"`
}

func fingerprintInput(input Input) legacyInput {
	return legacyInput(input)
}
