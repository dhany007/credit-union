package member

// Input contains the membership registration fields.
type Input struct {
	NIK     string `json:"nik"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

// Result contains the public member operation result.
type Result struct {
	ID        int64  `json:"id"`
	Number    string `json:"number"`
	Reference string `json:"reference"`
	Balance   int64  `json:"balance"`
}

// Member is the masked staff-facing membership projection.
type Member struct {
	ID           int64  `json:"id"`
	Number       string `json:"number"`
	Name         string `json:"name"`
	NIKMasked    string `json:"nik_masked"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	HomeBranchID int    `json:"home_branch_id"`
	HomeBranch   string `json:"home_branch"`
	Balance      int64  `json:"balance"`
}
