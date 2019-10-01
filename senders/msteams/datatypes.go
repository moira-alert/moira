package msteams

/*
Represents a Fact in a MessageCard, contains a timestamp and trigger data
 {
		"name": "10:45",
    "value": "someServer = 0.11 (NODATA to WARN)"
 }
*/
type Fact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

/*
Represents a Section in a MessageCard, contains Facts and the Trigger description
 {
		"activityTitle": "Description",
		"activityText": "A trigger description",
    "facts": [
			 {
					"name": "10:45",
					"value": "someServer = 0.11 (NODATA to WARN)"
			 }
		]
 }
*/
type Section struct {
	ActivityTitle string `json:"activityTitle"`
	ActivityText  string `json:"activityText"`
	Facts         []Fact `json:"facts"`
}

/*
Represents an OpenURITarget data in a MessageCard,creates a clickable target back to the trigger URI
 {
		"os": "default",
    "value": "http://moira.tld/trigger/ABCDEF-GH"
 }
*/
type OpenURITarget struct {
	Os  string `json:"os"`
	URI string `json:"uri"`
}

/*
Represents possible actions in a MessageCard, limited to OpenURI actions
 {
		"@type": "OpenUri",
    "name": "Open in Moira"
		"targets": [
			{
				"os": "default",
				"value": "http://moira.tld/trigger/ABCDEF-GH"
 			}
		]
 }
*/
type Actions struct {
	Type    string          `json:"@type"`
	Name    string          `json:"name"`
	Targets []OpenURITarget `json:"targets"`
}

/*
Represents an MSTeams compatible MessageCard
 {
		"@context": "https://schema.org/extensions",
    "@type": "MessageCard",
		"summary": "Moira Alert"
		"title" : "WARN Trigger Name [tag1]"
		"themeColor": "ffa500"
		"sections": [
			 {
					"activityTitle": "Description",
					"activityText": "A trigger description",
					"facts": [
						 {
								"name": "10:45",
								"value": "someServer = 0.11 (NODATA to WARN)"
						 }
					]
			 }
		]
		"potentialAction": [
			{
				"@type": "OpenUri",
				"name": "Open in Moira"
				"targets": [
					{
						"os": "default",
						"value": "http://moira.tld/trigger/ABCDEF-GH"
					}
				]
			}
		]
 }
*/
type MessageCard struct {
	Context         string    `json:"@context"`
	MessageType     string    `json:"@type"`
	Summary         string    `json:"summary"`
	ThemeColor      string    `json:"themeColor"`
	Title           string    `json:"title"`
	Sections        []Section `json:"sections"`
	PotentialAction []Actions `json:"potentialAction,omitempty"`
}
