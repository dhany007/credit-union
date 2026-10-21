package member

// Legacy projections preserve persisted request hashes across DTO refactors.

type legacyInput struct {
	NIK     string `json:"nik"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

func fingerprintInput(input Input) legacyInput {
	return legacyInput(input)
}
