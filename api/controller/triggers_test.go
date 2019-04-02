package controller

import (
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success with trigger.ID empty", t, func() {
		triggerModel := dto.TriggerModel{}
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), gomock.Any()).Return(nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger created")
	})

	Convey("Success with triggerID", t, func() {
		triggerID := uuid.Must(uuid.NewV4()).String()
		triggerModel := dto.TriggerModel{ID: triggerID}
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(moira.Trigger{}, database.ErrNil)
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), triggerModel.ToMoiraTrigger()).Return(nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger created")
		So(resp.ID, ShouldResemble, triggerID)
	})

	Convey("Trigger already exists", t, func() {
		triggerModel := dto.TriggerModel{ID: uuid.Must(uuid.NewV4()).String()}
		trigger := triggerModel.ToMoiraTrigger()
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(*trigger, nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger with this ID already exists")))
		So(resp, ShouldBeNil)
	})

	Convey("Get trigger error", t, func() {
		trigger := dto.TriggerModel{ID: uuid.Must(uuid.NewV4()).String()}
		expected := fmt.Errorf("soo bad trigger")
		dataBase.EXPECT().GetTrigger(trigger.ID).Return(moira.Trigger{}, expected)
		resp, err := CreateTrigger(dataBase, &trigger, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})

	Convey("Error", t, func() {
		triggerModel := dto.TriggerModel{ID: uuid.Must(uuid.NewV4()).String()}
		expected := fmt.Errorf("soo bad trigger")
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(moira.Trigger{}, database.ErrNil)
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 10)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), triggerModel.ToMoiraTrigger()).Return(expected)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})
}

func TestGetAllTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Has triggers", t, func() {
		triggerIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		triggers := []*moira.TriggerCheck{{Trigger: moira.Trigger{ID: triggerIDs[0]}}, {Trigger: moira.Trigger{ID: triggerIDs[1]}}}
		triggersList := []moira.TriggerCheck{{Trigger: moira.Trigger{ID: triggerIDs[0]}}, {Trigger: moira.Trigger{ID: triggerIDs[1]}}}
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggers, nil)
		list, err := GetAllTriggers(mockDatabase)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: triggersList})
	})

	Convey("No triggers", t, func() {
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(make([]string, 0), nil)
		mockDatabase.EXPECT().GetTriggerChecks(make([]string, 0)).Return(make([]*moira.TriggerCheck, 0), nil)
		list, err := GetAllTriggers(mockDatabase)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: make([]moira.TriggerCheck, 0)})
	})

	Convey("GetTriggerIDs error", t, func() {
		expected := fmt.Errorf("getTriggerIDs error")
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(nil, expected)
		list, err := GetAllTriggers(mockDatabase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("GetTriggerChecks error", t, func() {
		expected := fmt.Errorf("getTriggerChecks error")
		mockDatabase.EXPECT().GetAllTriggerIDs().Return(make([]string, 0), nil)
		mockDatabase.EXPECT().GetTriggerChecks(make([]string, 0)).Return(nil, expected)
		list, err := GetAllTriggers(mockDatabase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}

func TestSearchTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockIndex := mock_moira_alert.NewMockSearcher(mockCtrl)
	var page int64
	var size int64 = 50
	var exp int64 = 31
	triggerSearchResults := make([]*moira.SearchResult, 0)
	for _, triggerCheck := range triggerChecks {
		triggerSearchResults = append(triggerSearchResults, &moira.SearchResult{
			ObjectID:   triggerCheck.ID,
			Highlights: highLights,
		})
	}
	triggerIDs := make([]string, len(triggerChecks))
	triggersPointers := make([]*moira.TriggerCheck, len(triggerChecks))
	for i, trigger := range triggerChecks {
		newTrigger := new(moira.TriggerCheck)
		*newTrigger = trigger
		triggersPointers[i] = newTrigger
		triggerIDs[i] = trigger.ID
	}

	tags := make([]string, 0)
	searchString := ""

	Convey("No tags, no text, onlyErrors = false, ", t, func() {
		Convey("Page is bigger than triggers number", func() {
			mockIndex.EXPECT().SearchTriggers(tags, searchString, false, page, size).Return(triggerSearchResults, exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, page, size, false, tags, searchString)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks,
				Total: &exp,
				Page:  &page,
				Size:  &size,
			})
		})

		Convey("Must return all triggers, when size is -1", func() {
			size = -1
			mockIndex.EXPECT().SearchTriggers(tags, searchString, false, page, size).Return(triggerSearchResults, exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, page, size, false, tags, searchString)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks,
				Total: &exp,
				Page:  &page,
				Size:  &size,
			})
		})

		Convey("Page is less than triggers number", func() {
			size = 10
			mockIndex.EXPECT().SearchTriggers(tags, searchString, false, page, size).Return(triggerSearchResults[:10], exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[:10]).Return(triggersPointers[:10], nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, page, size, false, tags, searchString)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks[:10],
				Total: &exp,
				Page:  &page,
				Size:  &size,
			})

			Convey("Second page", func() {
				page = 1
				mockIndex.EXPECT().SearchTriggers(tags, searchString, false, page, size).Return(triggerSearchResults[10:20], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[10:20]).Return(triggersPointers[10:20], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, page, size, false, tags, searchString)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[10:20],
					Total: &exp,
					Page:  &page,
					Size:  &size,
				})
			})
		})
	})

	Convey("Complex search query", t, func() {
		size = 10
		page = 0
		Convey("Only errors", func() {
			exp = 30
			// superTrigger31 is the only trigger without errors
			mockIndex.EXPECT().SearchTriggers(tags, searchString, true, page, size).Return(triggerSearchResults[:10], exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[:10]).Return(triggersPointers[:10], nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, page, size, true, tags, searchString)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks[0:10],
				Total: &exp,
				Page:  &page,
				Size:  &size,
			})

			Convey("Only errors with tags", func() {
				tags = []string{"encounters", "Kobold"}
				exp = 2
				mockIndex.EXPECT().SearchTriggers(tags, searchString, true, page, size).Return(triggerSearchResults[1:3], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[1:3]).Return(triggersPointers[1:3], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, page, size, true, tags, searchString)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[1:3],
					Total: &exp,
					Page:  &page,
					Size:  &size,
				})
			})

			Convey("Only errors with text terms", func() {
				searchString = "dragonshield medium"
				exp = 1
				mockIndex.EXPECT().SearchTriggers(tags, searchString, true, page, size).Return(triggerSearchResults[2:3], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[2:3]).Return(triggersPointers[2:3], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, page, size, true, tags, searchString)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[2:3],
					Total: &exp,
					Page:  &page,
					Size:  &size,
				})
			})

			Convey("Only errors with tags and text terms", func() {
				tags = []string{"traps"}
				searchString = "deadly"
				exp = 4

				deadlyTraps := []moira.TriggerCheck{
					triggerChecks[10],
					triggerChecks[14],
					triggerChecks[18],
					triggerChecks[19],
				}

				deadlyTrapsPointers := []*moira.TriggerCheck{
					&triggerChecks[10],
					&triggerChecks[14],
					&triggerChecks[18],
					&triggerChecks[19],
				}

				deadlyTrapsTriggerIDs := make([]string, 0)
				deadlyTrapsSearchResults := make([]*moira.SearchResult, 0)
				for _, deadlyTrap := range deadlyTraps {
					deadlyTrapsSearchResults = append(deadlyTrapsSearchResults, &moira.SearchResult{
						ObjectID:   deadlyTrap.ID,
						Highlights: deadlyTrap.Highlights,
					})
					deadlyTrapsTriggerIDs = append(deadlyTrapsTriggerIDs, deadlyTrap.ID)
				}

				mockIndex.EXPECT().SearchTriggers(tags, searchString, true, page, size).Return(deadlyTrapsSearchResults, exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(deadlyTrapsTriggerIDs).Return(deadlyTrapsPointers, nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, page, size, true, tags, searchString)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  deadlyTraps,
					Total: &exp,
					Page:  &page,
					Size:  &size,
				})
			})
		})
	})

	Convey("Find triggers errors", t, func() {
		tags = make([]string, 0)
		searchString = ""

		Convey("Error from searcher", func() {
			searcherError := fmt.Errorf("very bad request")
			mockIndex.EXPECT().SearchTriggers(tags, searchString, false, page, size).Return(make([]*moira.SearchResult, 0), int64(0), searcherError)
			list, err := SearchTriggers(mockDatabase, mockIndex, page, size, false, tags, searchString)
			So(err, ShouldNotBeNil)
			So(list, ShouldBeNil)
		})

		Convey("Error from database", func() {
			size = 50
			searcherError := fmt.Errorf("very bad request")
			mockIndex.EXPECT().SearchTriggers(tags, searchString, false, page, size).Return(triggerSearchResults, exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(nil, searcherError)
			list, err := SearchTriggers(mockDatabase, mockIndex, page, size, false, tags, searchString)
			So(err, ShouldNotBeNil)
			So(list, ShouldBeNil)
		})
	})
}

var triggerChecks = []moira.TriggerCheck{
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger1",
			Name: "I used D&D character generator for test triggers: https://donjon.bin.sh",
			Tags: []string{"DND-generator", "common"},
		},
		LastCheck: moira.CheckData{
			Score: 30,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger2",
			Name: "Kobold Scale Sorcerer (cr 1, vgm 167) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Tags: []string{"DND-generator", "Kobold", "Sorcerer", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 29,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger3",
			Name: "Kobold Dragonshield (cr 1, vgm 165) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Tags: []string{"DND-generator", "Kobold", "Dragonshield", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 28,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger4",
			Name: "Orc Nurtured One of Yurtrus (cr 1/2, vgm 184) and 1 x Orc (cr 1/2, mm 246); hard, 200 xp",
			Tags: []string{"DND-generator", "Orc", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 27,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger5",
			Name: "Rust Monster (cr 1/2, mm 262); easy, 100 xp",
			Tags: []string{"DND-generator", "Rust-Monster", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 26,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger6",
			Name: "Giant Constrictor Snake (cr 2, mm 324); deadly, 450 xp",
			Tags: []string{"Giant", "DND-generator", "Snake", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 25,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger7",
			Name: "Darkling (cr 1/2, vgm 134); hard, 200 xp",
			Tags: []string{"Darkling", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 24,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger8",
			Name: "Ghost (cr 4, mm 147); hard, 1100 xp",
			Tags: []string{"Ghost", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 23,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger9",
			Name: "Spectator (cr 3, mm 30); medium, 700 xp",
			Tags: []string{"Spectator", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 22,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger10",
			Name: "Gibbering Mouther (cr 2, mm 157); easy, 450 xp",
			Tags: []string{"Gibbering-Mouther", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 21,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger11",
			Name: "Scythe Blade: DC 10 to find, DC 10 to disable; +11 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Tags: []string{"Scythe Blade", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 20,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger12",
			Name: "Falling Block: DC 10 to find, DC 10 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags: []string{"Falling-Block", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 19,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger13",
			Name: "Thunderstone Mine: DC 15 to find, DC 15 to disable; affects all targets within 20 ft., DC 15 save or take 2d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, dangerous",
			Tags: []string{"Thunderstone-Mine", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 18,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger14",
			Name: "Falling Block: DC 10 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags: []string{"Falling-Block", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 17,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger15",
			Name: "Chain Flail: DC 15 to find, DC 10 to disable; initiative +3, 1 attack per round, +11 to hit against all targets within 5 ft., 4d10 bludgeoning damage; apprentice tier, deadly",
			Tags: []string{"Chain-Flail", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 16,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger16",
			Name: "Falling Block: DC 15 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags: []string{"Falling-Block", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 15,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger17",
			Name: "Electrified Floortile: DC 20 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 15 save or take 2d10 lightning damage; apprentice tier, dangerous",
			Tags: []string{"Electrified-Floortile", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 14,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger18",
			Name: "Earthmaw Trap: DC 15 to find, DC 10 to disable; +7 to hit against one target, 2d10 piercing damage; apprentice tier, dangerous",
			Tags: []string{"Earthmaw-Trap", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 13,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger19",
			Name: "Thunderstone Mine: DC 15 to find, DC 20 to disable; affects all targets within 20 ft., DC 18 save or take 4d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, deadly",
			Tags: []string{"Thunderstone-Mine", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 12,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger20",
			Name: "Scythe Blade: DC 15 to find, DC 10 to disable; +12 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Tags: []string{"Scythe-Blade", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 11,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger21",
			Name: "Keelte: Female Elf Monk, LG. Str 12, Dex 14, Con 13, Int 9, Wis 15, Cha 14",
			Tags: []string{"Female", "DND-generator", "Elf", "Monk", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 10,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger22",
			Name: "Kather Larke: Female Halfling Cleric, CN. Str 8, Dex 8, Con 13, Int 7, Wis 13, Cha 10",
			Tags: []string{"Female", "DND-generator", "Halfling", "Cleric", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 9,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger23",
			Name: "Cyne: Male Human Soldier, NG. Str 12, Dex 9, Con 8, Int 10, Wis 8, Cha 10",
			Tags: []string{"Male", "DND-generator", "Human", "Soldier", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 8,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger24",
			Name: "Gytha: Female Human Barbarian, N. Str 16, Dex 13, Con 12, Int 12, Wis 14, Cha 9",
			Tags: []string{"Female", "DND-generator", "Human", "Barbarian", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 7,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger25",
			Name: "Brobern Hawte: Male Half-elf Monk, N. Str 12, Dex 10, Con 8, Int 14, Wis 12, Cha 12",
			Tags: []string{"Male", "DND-generator", "Half-elf", "Monk", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 6,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger26",
			Name: "Borneli: Male Elf Servant, LN. Str 12, Dex 12, Con 8, Int 13, Wis 6, Cha 12",
			Tags: []string{"Male", "DND-generator", "Elf", "Servant", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 5,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger27",
			Name: "Midda: Male Elf Sorcerer, LN. Str 10, Dex 13, Con 11, Int 7, Wis 10, Cha 13",
			Tags: []string{"Male", "DND-generator", "Elf", "Sorcerer", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 4,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger28",
			Name: "Burgwe: Female Human Bard, CN. Str 13, Dex 11, Con 10, Int 13, Wis 12, Cha 17.",
			Tags: []string{"Female", "DND-generator", "Human", "Bard", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 3,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger29",
			Name: "Carel: Female Gnome Druid, Neutral. Str 11, Dex 12, Con 7, Int 10, Wis 17, Cha 10",
			Tags: []string{"Female", "DND-generator", "Gnome", "Druid", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 2,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger30",
			Name: "Suse Salte: Female Human Aristocrat, N. Str 10, Dex 7, Con 10, Int 9, Wis 7, Cha 13",
			Tags: []string{"Female", "DND-generator", "Human", "Aristocrat", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 1,
		},
		Highlights: highLights,
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger31",
			Name: "Surprise!",
			Tags: []string{"Something-extremely-new"},
		},
		LastCheck: moira.CheckData{
			Score: 0,
		},
		Highlights: highLights,
	},
}

var highLights = []moira.SearchHighlight{
	{
		Field: "testField",
		Value: "testValue",
	},
}
