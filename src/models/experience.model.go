package userModel

type Experience struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Body      string   `json:"body"`
	ImageURLs []string `json:"imageUrls"`
	// ThumbURL is only filled by the cards view (response-only, never persisted).
	ThumbURL   string   `json:"thumbUrl,omitempty" dynamodbav:"-"`
	Tags       []string `json:"tags"`
	Visibility string   `json:"visibility"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}
