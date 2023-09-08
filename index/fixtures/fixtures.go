package fixtures

import "github.com/moira-alert/moira"

type fixtureIndexedField struct {
	content    string
	highlights map[string][]moira.SearchHighlight
}

type fixtureIndexedTriggers struct {
	list []fixtureIndexedTrigger
}

type fixtureIndexedTrigger struct {
	triggerID        string
	triggerName      fixtureIndexedField
	triggerDesc      fixtureIndexedField
	triggerTags      []string
	triggerCreatedBy string
	triggerScore     int64
}

func (it *fixtureIndexedTrigger) GetHighLights(searchString string) []moira.SearchHighlight {
	highlights := make([]moira.SearchHighlight, 0)
	if nameHighlights, ok := it.triggerName.highlights[searchString]; ok {
		highlights = append(highlights, nameHighlights...)
	}
	if descHighlights, ok := it.triggerDesc.highlights[searchString]; ok {
		highlights = append(highlights, descHighlights...)
	}
	return highlights
}

func (its *fixtureIndexedTriggers) ToTriggerChecks() []*moira.TriggerCheck {
	triggerChecks := make([]*moira.TriggerCheck, 0)
	for _, indexedTrigger := range its.list {
		triggerCheck := moira.TriggerCheck{
			Trigger: moira.Trigger{
				ID:        indexedTrigger.triggerID,
				Name:      indexedTrigger.triggerName.content,
				Tags:      indexedTrigger.triggerTags,
				CreatedBy: indexedTrigger.triggerCreatedBy,
				Desc:      new(string),
			},
			LastCheck: moira.CheckData{
				Score: indexedTrigger.triggerScore,
			},
		}
		*triggerCheck.Trigger.Desc = indexedTrigger.triggerDesc.content
		triggerChecks = append(triggerChecks, &triggerCheck)
	}
	return triggerChecks
}

func (its *fixtureIndexedTriggers) ToSearchResults(searchString string) []*moira.SearchResult {
	searchResults := make([]*moira.SearchResult, 0)
	for _, indexedTrigger := range its.list {
		searchResult := moira.SearchResult{
			ObjectID:   indexedTrigger.triggerID,
			Highlights: indexedTrigger.GetHighLights(searchString),
		}
		searchResults = append(searchResults, &searchResult)
	}
	return searchResults
}

func (its *fixtureIndexedTriggers) ToTriggerIDs() []string {
	triggerIDs := make([]string, 0)
	for _, indexedTrigger := range its.list {
		triggerIDs = append(triggerIDs, indexedTrigger.triggerID)
	}
	return triggerIDs
}

// IndexedTriggerTestCases is a fixture to test fulltext search
var IndexedTriggerTestCases = fixtureIndexedTriggers{
	list: []fixtureIndexedTrigger{
		{
			triggerID: "SuperTrigger1",
			triggerName: fixtureIndexedField{
				content: "I used D&D character generator for test triggers: https://donjon.bin.sh",
			},
			triggerDesc: fixtureIndexedField{
				content: "0: Is this the real life? Is this just fantasy?",
			},
			triggerTags:      []string{"DND-generator", "common"},
			triggerCreatedBy: "test",
			triggerScore:     30, //nolint
		},
		{
			triggerID: "SuperTrigger2",
			triggerName: fixtureIndexedField{
				content: "Kobold Scale Sorcerer (cr 1, vgm 167) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			},
			triggerDesc: fixtureIndexedField{
				content: "1: Caught in a landslide, no escape from reality",
			},
			triggerTags:      []string{"DND-generator", "Kobold", "Sorcerer", "encounters"},
			triggerCreatedBy: "test",
			triggerScore:     29, //nolint
		},
		{
			triggerID: "SuperTrigger3",
			triggerName: fixtureIndexedField{
				content: "Kobold Dragonshield (cr 1, vgm 165) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
				highlights: map[string][]moira.SearchHighlight{
					"dragonshield medium": {
						{
							Field: "name",
							Value: "Kobold <mark>Dragonshield</mark> (cr 1, vgm 165) and 1 x Kobold (cr 1/8, mm 195); <mark>medium</mark>, 225 xp",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: "2: Open your eyes, look up to the skies and see",
			},
			triggerTags:      []string{"DND-generator", "Kobold", "Dragonshield", "encounters"},
			triggerCreatedBy: "test",
			triggerScore:     28, //nolint
		},
		{
			triggerID: "SuperTrigger4",
			triggerName: fixtureIndexedField{
				content: "Orc Nurtured One of Yurtrus (cr 1/2, vgm 184) and 1 x Orc (cr 1/2, mm 246); hard, 200 xp",
			},
			triggerDesc: fixtureIndexedField{
				content: "3: I'm just a poor boy, I need no sympathy",
			},
			triggerTags:      []string{"DND-generator", "Orc", "encounters"},
			triggerCreatedBy: "test",
			triggerScore:     27, //nolint
		},
		{
			triggerID: "SuperTrigger5",
			triggerName: fixtureIndexedField{
				content: "Rust Monster (cr 1/2, mm 262); easy, 100 xp",
				highlights: map[string][]moira.SearchHighlight{
					"easy": {
						{
							Field: "name",
							Value: "Rust Monster (cr 1/2, mm 262); <mark>easy</mark>, 100 xp",
						},
					},
					"little monster": {
						{
							Field: "name",
							Value: "Rust <mark>Monster</mark> (cr 1/2, mm 262); easy, 100 xp",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: "4: Because I'm easy come, easy go, little high, little low",
				highlights: map[string][]moira.SearchHighlight{
					"easy": {
						{
							Field: "desc",
							Value: "4: Because I&#39;m <mark>easy</mark> come, <mark>easy</mark> go, little high, little low",
						},
					},
					"little monster": {
						{
							Field: "desc",
							Value: "4: Because I&#39;m easy come, easy go, <mark>little</mark> high, <mark>little</mark> low",
						},
					},
				},
			},
			triggerTags:  []string{"DND-generator", "Rust-Monster", "encounters"},
			triggerScore: 26, //nolint
		},
		{
			triggerID: "SuperTrigger6",
			triggerName: fixtureIndexedField{
				content: "Giant Constrictor Snake (cr 2, mm 324); deadly, 450 xp",
			},
			triggerDesc: fixtureIndexedField{
				content: "5: Any way the wind blows doesn't really matter to me, to me",
			},
			triggerTags:  []string{"Giant", "DND-generator", "Snake", "encounters", "Darkness"},
			triggerScore: 25, //nolint
		},
		{
			triggerID: "SuperTrigger7",
			triggerName: fixtureIndexedField{
				content: "Darkling (cr 1/2, vgm 134); hard, 200 xp",
			},
			triggerDesc: fixtureIndexedField{
				content: "6",
			},
			triggerTags:  []string{"Darkling", "DND-generator", "encounters", "Darkness"},
			triggerScore: 24, //nolint
		},
		{
			triggerID: "SuperTrigger8",
			triggerName: fixtureIndexedField{
				content: "Ghost (cr 4, mm 147); hard, 1100 xp",
			},
			triggerDesc: fixtureIndexedField{
				content: "7: Mama, just killed a man",
			},
			triggerTags:      []string{"Ghost", "DND-generator", "encounters"},
			triggerCreatedBy: "monster",
			triggerScore:     23, //nolint
		},
		{
			triggerID: "SuperTrigger9",
			triggerName: fixtureIndexedField{
				content: "Spectator (cr 3, mm 30); medium, 700 xp",
			},
			triggerDesc: fixtureIndexedField{
				content: "8: Put a gun against his head, pulled my trigger, now he's dead",
			},
			triggerTags:      []string{"Spectator", "DND-generator", "encounters"},
			triggerCreatedBy: "monster",
			triggerScore:     22, //nolint
		},
		{
			triggerID: "SuperTrigger10",
			triggerName: fixtureIndexedField{
				content: "Gibbering Mouther (cr 2, mm 157); easy, 450 xp",
				highlights: map[string][]moira.SearchHighlight{
					"easy": {
						{
							Field: "name",
							Value: "Gibbering Mouther (cr 2, mm 157); <mark>easy</mark>, 450 xp",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: "9: Mama, life had just begun",
			},
			triggerTags:      []string{"Gibbering-Mouther", "DND-generator", "encounters"},
			triggerCreatedBy: "monster",
			triggerScore:     21, //nolint
		},
		{
			triggerID: "SuperTrigger11",
			triggerName: fixtureIndexedField{
				content: "Scythe Blade: DC 10 to find, DC 10 to disable; +11 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
				highlights: map[string][]moira.SearchHighlight{
					"deadly": {
						{
							Field: "name",
							Value: "Scythe Blade: DC 10 to find, DC 10 to disable; +11 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, <mark>deadly</mark>",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: "10: But now I've gone and thrown it all away",
			},
			triggerTags:      []string{"Scythe Blade", "DND-generator", "traps"},
			triggerCreatedBy: "monster",
			triggerScore:     20, //nolint
		},
		{
			triggerID: "SuperTrigger12",
			triggerName: fixtureIndexedField{
				content: "Falling Block: DC 10 to find, DC 10 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			},
			triggerDesc: fixtureIndexedField{
				content: "11: Mama, ooh, didn't mean to make you cry",
				highlights: map[string][]moira.SearchHighlight{
					"mama": {
						{
							Field: "desc",
							Value: "11: <mark>Mama</mark>, ooh, didn&#39;t mean to make you cry",
						},
					},
				},
			},
			triggerTags:      []string{"Falling-Block", "DND-generator", "traps"},
			triggerCreatedBy: "monster",
			triggerScore:     19, //nolint
		},
		{
			triggerID: "SuperTrigger13",
			triggerName: fixtureIndexedField{
				content: "Thunderstone Mine: DC 15 to find, DC 15 to disable; affects all targets within 20 ft., DC 15 save or take 2d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, dangerous",
			},
			triggerDesc: fixtureIndexedField{
				content: "12: If I'm not back again this time tomorrow",
			},
			triggerTags:      []string{"Thunderstone-Mine", "DND-generator", "traps"},
			triggerCreatedBy: "monster",
			triggerScore:     18, //nolint
		},
		{
			triggerID: "SuperTrigger14",
			triggerName: fixtureIndexedField{
				content: "Falling Block: DC 10 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			},
			triggerDesc: fixtureIndexedField{
				content: "13: Carry on, carry on as if nothing really matters",
			},
			triggerTags:      []string{"Falling-Block", "DND-generator", "traps"},
			triggerCreatedBy: "monster",
			triggerScore:     17, //nolint
		},
		{
			triggerID: "SuperTrigger15",
			triggerName: fixtureIndexedField{
				content: "Chain Flail: DC 15 to find, DC 10 to disable; initiative +3, 1 attack per round, +11 to hit against all targets within 5 ft., 4d10 bludgeoning damage; apprentice tier, deadly",
				highlights: map[string][]moira.SearchHighlight{
					"deadly": {
						{
							Field: "name",
							Value: "Chain Flail: DC 15 to find, DC 10 to disable; initiative +3, 1 attack per round, +11 to hit against all targets within 5 ft., 4d10 bludgeoning damage; apprentice tier, <mark>deadly</mark>",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: "14",
			},
			triggerTags:      []string{"Chain-Flail", "DND-generator", "traps", "shadows"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     16, //nolint
		},
		{
			triggerID: "SuperTrigger16",
			triggerName: fixtureIndexedField{
				content: "Falling Block: DC 15 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			},
			triggerDesc: fixtureIndexedField{
				content: "15: Too late, my time has come",
			},
			triggerTags:      []string{"Falling-Block", "DND-generator", "traps", "shadows"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     15, //nolint
		},
		{
			triggerID: "SuperTrigger17",
			triggerName: fixtureIndexedField{
				content: "Electrified Floortile: DC 20 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 15 save or take 2d10 lightning damage; apprentice tier, dangerous",
			},
			triggerDesc: fixtureIndexedField{
				content: "16: Sends shivers down my spine, body's aching all the time",
			},
			triggerTags:      []string{"Electrified-Floortile", "DND-generator", "traps", "Coldness", "Dark"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     14, //nolint
		},
		{
			triggerID: "SuperTrigger18",
			triggerName: fixtureIndexedField{
				content: "Earthmaw Trap: DC 15 to find, DC 10 to disable; +7 to hit against one target, 2d10 piercing damage; apprentice tier, dangerous",
			},
			triggerDesc: fixtureIndexedField{
				content: "17: Goodbye, everybody, I've got to go",
			},
			triggerTags:      []string{"Earthmaw-Trap", "DND-generator", "traps", "Coldness", "Dark"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     13, //nolint
		},
		{
			triggerID: "SuperTrigger19",
			triggerName: fixtureIndexedField{
				content: "Thunderstone Mine: DC 15 to find, DC 20 to disable; affects all targets within 20 ft., DC 18 save or take 4d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, deadly",
				highlights: map[string][]moira.SearchHighlight{
					"deadly": {
						{
							Field: "name",
							Value: "Thunderstone Mine: DC 15 to find, DC 20 to disable; affects all targets within 20 ft., DC 18 save or take 4d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, <mark>deadly</mark>",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: "18: Gotta leave you all behind and face the truth",
			},
			triggerTags:      []string{"Thunderstone-Mine", "DND-generator", "traps", "Coldness", "Dark"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     12, //nolint
		},
		{
			triggerID: "SuperTrigger20",
			triggerName: fixtureIndexedField{
				content: "Scythe Blade: DC 15 to find, DC 10 to disable; +12 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
				highlights: map[string][]moira.SearchHighlight{
					"deadly": {
						{
							Field: "name",
							Value: "Scythe Blade: DC 15 to find, DC 10 to disable; +12 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, <mark>deadly</mark>",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: "19: Mama, ooh, (Any way the wind blows)",
				highlights: map[string][]moira.SearchHighlight{
					"mama": {
						{
							Field: "desc",
							Value: "19: <mark>Mama</mark>, ooh, (Any way the wind blows)",
						},
					},
				},
			},
			triggerTags:      []string{"Scythe-Blade", "DND-generator", "traps"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     11, //nolint
		},
		{
			triggerID: "SuperTrigger21",
			triggerName: fixtureIndexedField{
				content: "Keelte: Female Elf Monk, LG. Str 12, Dex 14, Con 13, Int 9, Wis 15, Cha 14",
			},
			triggerDesc: fixtureIndexedField{
				content: "20: I don't wanna die",
			},
			triggerTags:      []string{"Female", "DND-generator", "Elf", "Monk", "NPCs"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     10, //nolint
		},
		{
			triggerID: "SuperTrigger22",
			triggerName: fixtureIndexedField{
				content: "Kather Larke: Female Halfling Cleric, CN. Str 8, Dex 8, Con 13, Int 7, Wis 13, Cha 10",
			},
			triggerDesc: fixtureIndexedField{
				content: "21: I sometimes wish I'd never been born at all",
			},
			triggerTags:      []string{"Female", "DND-generator", "Halfling", "Cleric", "NPCs"},
			triggerCreatedBy: "tarasov.da",
			triggerScore:     9, //nolint
		},
		{
			triggerID: "SuperTrigger23",
			triggerName: fixtureIndexedField{
				content: "Cyne: Male Human Soldier, NG. Str 12, Dex 9, Con 8, Int 10, Wis 8, Cha 10",
			},
			triggerDesc: fixtureIndexedField{
				content: "22",
			},
			triggerTags:      []string{"Male", "DND-generator", "Human", "Soldier", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     8, //nolint
		},
		{
			triggerID: "SuperTrigger24",
			triggerName: fixtureIndexedField{
				content: "Gytha: Female Human Barbarian, N. Str 16, Dex 13, Con 12, Int 12, Wis 14, Cha 9",
			},
			triggerDesc: fixtureIndexedField{
				content: "23: I see a little silhouetto of a man",
			},
			triggerTags:      []string{"Female", "DND-generator", "Human", "Barbarian", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     7, //nolint
		},
		{
			triggerID: "SuperTrigger25",
			triggerName: fixtureIndexedField{
				content: "Brobern Hawte: Male Half-elf Monk, N. Str 12, Dex 10, Con 8, Int 14, Wis 12, Cha 12",
			},
			triggerDesc: fixtureIndexedField{
				content: "24: Scaramouche, Scaramouche, will you do the Fandango?",
			},
			triggerTags:      []string{"Male", "DND-generator", "Half-elf", "Monk", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     6, //nolint
		},
		{
			triggerID: "SuperTrigger26",
			triggerName: fixtureIndexedField{
				content: "Borneli: Male Elf Servant, LN. Str 12, Dex 12, Con 8, Int 13, Wis 6, Cha 12",
			},
			triggerDesc: fixtureIndexedField{
				content: "25: Thunderbolt and lightning, very, very fright'ning me",
			},
			triggerTags:      []string{"Male", "DND-generator", "Elf", "Servant", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     5, //nolint
		},
		{
			triggerID: "SuperTrigger27",
			triggerName: fixtureIndexedField{
				content: "Midda: Male Elf Sorcerer, LN. Str 10, Dex 13, Con 11, Int 7, Wis 10, Cha 13",
			},
			triggerDesc: fixtureIndexedField{
				content: "26: (Galileo) Galileo, (Galileo) Galileo, Galileo Figaro magnifico",
			},
			triggerTags:      []string{"Male", "DND-generator", "Elf", "Sorcerer", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     4, //nolint
		},
		{
			triggerID: "SuperTrigger28",
			triggerName: fixtureIndexedField{
				content: "Burgwe: Female Human Bard, CN. Str 13, Dex 11, Con 10, Int 13, Wis 12, Cha 17.",
			},
			triggerDesc: fixtureIndexedField{
				content: "27: I'm just a poor boy, nobody loves me",
			},
			triggerTags:      []string{"Female", "DND-generator", "Human", "Bard", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     3, //nolint
		},
		{
			triggerID: "SuperTrigger29",
			triggerName: fixtureIndexedField{
				content: "Carel: Female Gnome Druid, Neutral. Str 11, Dex 12, Con 7, Int 10, Wis 17, Cha 10",
			},
			triggerDesc: fixtureIndexedField{
				content: "28: He's just a poor boy from a poor family",
			},
			triggerTags:      []string{"Female", "DND-generator", "Gnome", "Druid", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     2, //nolint
		},
		{
			triggerID: "SuperTrigger30",
			triggerName: fixtureIndexedField{
				content: "Suse Salte: Female Human Aristocrat, N. Str 10, Dex 7, Con 10, Int 9, Wis 7, Cha 13",
			},
			triggerDesc: fixtureIndexedField{
				content: "29: Spare him his life from this monstrosity",
			},
			triggerTags:      []string{"Female", "DND-generator", "Human", "Aristocrat", "NPCs"},
			triggerCreatedBy: "internship2023",
			triggerScore:     1,
		},
		{
			triggerID: "SuperTrigger31",
			triggerName: fixtureIndexedField{
				content: "Surprise easy!",
				highlights: map[string][]moira.SearchHighlight{
					"easy": {
						{
							Field: "name",
							Value: "Surprise <mark>easy</mark>!",
						},
					},
				},
			},
			triggerDesc: fixtureIndexedField{
				content: `30: Easy come, easy go, will you let me go?
				Bismillah! No, we will not let you go
				(Let him go!) Bismillah! We will not let you go
				(Let him go!) Bismillah! We will not let you go
				(Let me go) Will not let you go
				(Let me go) Will not let you go
				(Let me go) Ah
				No, no, no, no, no, no, no
				(Oh mamma mia, mamma mia) Mamma mia, let me go
				Beelzebub has a devil put aside for me, for me, for me!`,
				highlights: map[string][]moira.SearchHighlight{
					"easy": {
						{
							Field: "desc",
							Value: "…: <mark>Easy</mark> come, <mark>easy</mark> go, will you let me go?\n\t\t\t\tBismillah! No, we will not let you go\n\t\t\t\t(Let him go!) Bismillah! We will not let you go\n\t\t\t\t(Let him go!) Bismillah! We will not let you go\n\t\t\t\t(Let me …",
						},
					},
				},
			},
			triggerTags:      []string{"Something-extremely-new"},
			triggerCreatedBy: "internship2023",
			triggerScore:     0,
		},
		{
			triggerID: "SuperTrigger32",
			triggerName: fixtureIndexedField{
				content: "Surprise!",
			},
			triggerDesc: fixtureIndexedField{
				content: `30: Easy come, easy go, will you let me go?
				Bismillah! No, we will not let you go
				(Let him go!) Bismillah! We will not let you go
				(Let him go!) Bismillah! We will not let you go
				(Let me go) Will not let you go
				(Let me go) Will not let you go
				(Let me go) Ah
				No, no, no, no, no, no, no
				(Oh mamma mia, mamma mia) Mamma mia, let me go
				Beelzebub has a devil put aside for me, for me, for me!`,
				highlights: map[string][]moira.SearchHighlight{
					"easy": {
						{
							Field: "desc",
							Value: "…: <mark>Easy</mark> come, <mark>easy</mark> go, will you let me go?\n\t\t\t\tBismillah! No, we will not let you go\n\t\t\t\t(Let him go!) Bismillah! We will not let you go\n\t\t\t\t(Let him go!) Bismillah! We will not let you go\n\t\t\t\t(Let me …",
						},
					},
				},
			},
			triggerTags:      []string{"Something-extremely-new"},
			triggerCreatedBy: "internship2023",
			triggerScore:     0,
		},
	},
}
