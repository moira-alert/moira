package msteams

type Fact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Section struct {
	ActivityTitle string `json:"activityTitle"`
	ActivityText  string `json:"activityText"`
	Facts         []Fact `json:"facts"`
}

type OpenUriTarget struct {
	Os  string `json:"os"`
	Uri string `json:"uri"`
}

//limited to OpenURI actions
type Actions struct {
	Type    string          `json:"@type"`
	Name    string          `json:"name"`
	Targets []OpenUriTarget `json:"targets"`
}

type MessageCard struct {
	Context         string    `json:"@context"`
	MessageType     string    `json:"@type"`
	Summary         string    `json:"summary"`
	ThemeColor      string    `json:"themeColor"`
	Title           string    `json:"title"`
	Sections        []Section `json:"sections"`
	PotentialAction []Actions `json:"potentialAction,omitempty"`
}
