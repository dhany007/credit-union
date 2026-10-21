package auth

// Actor is the authenticated staff identity and servicing KCP.
type Actor struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	BranchID   int    `json:"branch_id"`
	BranchName string `json:"branch_name"`
}

// LoginInput contains staff credentials accepted by the login endpoint.
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
