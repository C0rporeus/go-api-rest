package userModel

type Experience struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Body       string   `json:"body"`
	Tags       []string `json:"tags"`
	Visibility string   `json:"visibility"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}
