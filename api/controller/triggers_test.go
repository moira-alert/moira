package controller

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/satori/go.uuid"
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
		triggerID := uuid.NewV4().String()
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
		triggerModel := dto.TriggerModel{ID: uuid.NewV4().String()}
		trigger := triggerModel.ToMoiraTrigger()
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(*trigger, nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInvalidRequest(fmt.Errorf("trigger with this ID already exists")))
		So(resp, ShouldBeNil)
	})

	Convey("Get trigger error", t, func() {
		trigger := dto.TriggerModel{ID: uuid.NewV4().String()}
		expected := fmt.Errorf("soo bad trigger")
		dataBase.EXPECT().GetTrigger(trigger.ID).Return(moira.Trigger{}, expected)
		resp, err := CreateTrigger(dataBase, &trigger, make(map[string]bool))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(resp, ShouldBeNil)
	})

	Convey("Error", t, func() {
		triggerModel := dto.TriggerModel{ID: uuid.NewV4().String()}
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
		triggerIDs := []string{uuid.NewV4().String(), uuid.NewV4().String()}
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

func TestGetTriggerIdsRange(t *testing.T) {
	triggers := make([]string, 20)
	for i := range triggers {
		triggers[i] = uuid.NewV4().String()
	}
	Convey("Has triggers in range", t, func() {
		expected := getTriggerIdsRange(triggers, 20, 3, 5)
		So(expected, ShouldResemble, triggers[15:20])

		expected = getTriggerIdsRange(triggers, 20, 2, 5)
		So(expected, ShouldResemble, triggers[10:15])
	})

	Convey("No triggers on range", t, func() {
		expected := getTriggerIdsRange(triggers, 20, 4, 5)
		So(expected, ShouldResemble, make([]string, 0))

		expected = getTriggerIdsRange(triggers, 20, 55, 1)
		So(expected, ShouldResemble, make([]string, 0))

		expected = getTriggerIdsRange(triggers, 20, 3, 10)
		So(expected, ShouldResemble, make([]string, 0))
	})

	Convey("Range takes part or triggers", t, func() {
		expected := getTriggerIdsRange(triggers, 20, 3, 6)
		So(expected, ShouldResemble, triggers[18:20])

		expected = getTriggerIdsRange(triggers, 20, 1, 11)
		So(expected, ShouldResemble, triggers[11:20])

		expected = getTriggerIdsRange(triggers, 20, 0, 30)
		So(expected, ShouldResemble, triggers)
	})
}

func TestGetTriggerPage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	var page int64
	var size int64 = 10
	triggerIDs := make([]string, 20)
	for i := range triggerIDs {
		triggerIDs[i] = uuid.NewV4().String()
	}
	triggers := make([]moira.TriggerCheck, 20)
	for i := range triggerIDs {
		triggers[i] = moira.TriggerCheck{Trigger: moira.Trigger{ID: triggerIDs[i]}}
	}
	triggersPointers := make([]*moira.TriggerCheck, 20)
	for i := range triggerIDs {
		triggersPointers[i] = &moira.TriggerCheck{Trigger: moira.Trigger{ID: triggerIDs[i]}}
	}

	Convey("Has tags and only errors", t, func() {
		tags := []string{"tag1", "tag2"}
		var exp int64 = 20
		mockDatabase.EXPECT().GetTriggerCheckIDs(tags, true).Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(triggersPointers[0:10], nil)
		list, err := GetTriggerPage(mockDatabase, page, size, true, tags)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{
			List:  triggers[0:10],
			Total: &exp,
			Page:  &page,
			Size:  &size,
		})
	})

	Convey("All triggers", t, func() {
		var exp int64 = 20
		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), false).Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(triggersPointers[0:10], nil)
		list, err := GetTriggerPage(mockDatabase, page, size, false, make([]string, 0))
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{
			List:  triggers[0:10],
			Total: &exp,
			Page:  &page,
			Size:  &size,
		})
	})

	Convey("Error GetFilteredTriggerCheckIDs", t, func() {
		expected := fmt.Errorf("getFilteredTriggerCheckIDs error")
		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), true).Return(nil, expected)
		list, err := GetTriggerPage(mockDatabase, 0, 20, true, make([]string, 0))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("Error GetTriggerChecks", t, func() {
		expected := fmt.Errorf("getTriggerChecks error")
		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), false).Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(nil, expected)
		list, err := GetTriggerPage(mockDatabase, page, size, false, make([]string, 0))
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})
}

//func TestFindTriggersPerPage(t *testing.T) {
//	mockCtrl := gomock.NewController(t)
//	defer mockCtrl.Finish()
//	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
//	//mockIndex := mock_moira_alert.NewMockSearcher(mockCtrl)
//	var page int64
//	var size int64 = 10
//	triggerIDs := make([]string, 20)
//	for i := range triggerIDs {
//		triggerIDs[i] = uuid.NewV4().String()
//	}
//	triggers := make([]moira.TriggerCheck, 20)
//	for i := range triggerIDs {
//		triggers[i] = moira.TriggerCheck{Trigger: moira.Trigger{ID: triggerIDs[i]}}
//	}
//	triggersPointers := make([]*moira.TriggerCheck, 20)
//	for i := range triggerIDs {
//		triggersPointers[i] = &moira.TriggerCheck{Trigger: moira.Trigger{ID: triggerIDs[i]}}
//	}
//
//	Convey("Has tags and only errors", t, func() {
//		tags := []string{"tag1", "tag2"}
//		var exp int64 = 20
//		mockDatabase.EXPECT().GetTriggerCheckIDs(tags, true).Return(triggerIDs, nil)
//		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(triggersPointers[0:10], nil)
//		list, err := GetTriggerPage(mockDatabase, page, size, true, tags)
//		So(err, ShouldBeNil)
//		So(list, ShouldResemble, &dto.TriggersList{
//			List:  triggers[0:10],
//			Total: &exp,
//			Page:  &page,
//			Size:  &size,
//		})
//	})
//
//	Convey("All triggers", t, func() {
//		var exp int64 = 20
//		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), false).Return(triggerIDs, nil)
//		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(triggersPointers[0:10], nil)
//		list, err := GetTriggerPage(mockDatabase, page, size, false, make([]string, 0))
//		So(err, ShouldBeNil)
//		So(list, ShouldResemble, &dto.TriggersList{
//			List:  triggers[0:10],
//			Total: &exp,
//			Page:  &page,
//			Size:  &size,
//		})
//	})
//
//	Convey("Error GetFilteredTriggerCheckIDs", t, func() {
//		expected := fmt.Errorf("getFilteredTriggerCheckIDs error")
//		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), true).Return(nil, expected)
//		list, err := GetTriggerPage(mockDatabase, 0, 20, true, make([]string, 0))
//		So(err, ShouldResemble, api.ErrorInternalServer(expected))
//		So(list, ShouldBeNil)
//	})
//
//	Convey("Error GetTriggerChecks", t, func() {
//		expected := fmt.Errorf("getTriggerChecks error")
//		mockDatabase.EXPECT().GetTriggerCheckIDs(make([]string, 0), false).Return(triggerIDs, nil)
//		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[0:10]).Return(nil, expected)
//		list, err := GetTriggerPage(mockDatabase, page, size, false, make([]string, 0))
//		So(err, ShouldResemble, api.ErrorInternalServer(expected))
//		So(list, ShouldBeNil)
//	})
//}

var triggerChecks = []*moira.TriggerCheck{
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger1",
			Name: "I used D&D character generator for test triggers: https://donjon.bin.sh",
			Tags: []string{"DND-generator", "common"},
		},
		LastCheck: moira.CheckData{
			Score: 0,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger2",
			Name: "Kobold Scale Sorcerer (cr 1, vgm 167) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Tags: []string{"DND-generator", "Kobold", "Sorcerer", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 1,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger3",
			Name: "Kobold Dragonshield (cr 1, vgm 165) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Tags: []string{"DND-generator", "Kobold", "Dragonshield", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 2,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger4",
			Name: "Orc Nurtured One of Yurtrus (cr 1/2, vgm 184) and 1 x Orc (cr 1/2, mm 246); hard, 200 xp",
			Tags: []string{"DND-generator", "Orc", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 3,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger5",
			Name: "Rust Monster (cr 1/2, mm 262); easy, 100 xp",
			Tags: []string{"DND-generator", "Rust-Monster", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 4,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger6",
			Name: "Giant Constrictor Snake (cr 2, mm 324); deadly, 450 xp",
			Tags: []string{"Giant", "DND-generator", "Snake", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 5,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger7",
			Name: "Darkling (cr 1/2, vgm 134); hard, 200 xp",
			Tags: []string{"Darkling", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 6,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger8",
			Name: "Ghost (cr 4, mm 147); hard, 1100 xp",
			Tags: []string{"Ghost", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 7,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger9",
			Name: "Spectator (cr 3, mm 30); medium, 700 xp",
			Tags: []string{"Spectator", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 8,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger10",
			Name: "Gibbering Mouther (cr 2, mm 157); easy, 450 xp",
			Tags: []string{"Gibbering-Mouther", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 9,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger11",
			Name: "Scythe Blade: DC 10 to find, DC 10 to disable; +11 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Tags: []string{"Scythe Blade", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 10,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger12",
			Name: "Falling Block: DC 10 to find, DC 10 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags: []string{"Falling-Block", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 11,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger13",
			Name: "Thunderstone Mine: DC 15 to find, DC 15 to disable; affects all targets within 20 ft., DC 15 save or take 2d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, dangerous",
			Tags: []string{"Thunderstone-Mine", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 12,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger14",
			Name: "Falling Block: DC 10 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags: []string{"Falling-Block", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 13,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger15",
			Name: "Chain Flail: DC 15 to find, DC 10 to disable; initiative +3, 1 attack per round, +11 to hit against all targets within 5 ft., 4d10 bludgeoning damage; apprentice tier, deadly",
			Tags: []string{"Chain-Flail", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 14,
		},
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
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger17",
			Name: "Electrified Floortile: DC 20 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 15 save or take 2d10 lightning damage; apprentice tier, dangerous",
			Tags: []string{"Electrified-Floortile", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 16,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger18",
			Name: "Earthmaw Trap: DC 15 to find, DC 10 to disable; +7 to hit against one target, 2d10 piercing damage; apprentice tier, dangerous",
			Tags: []string{"Earthmaw-Trap", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 17,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger19",
			Name: "Thunderstone Mine: DC 15 to find, DC 20 to disable; affects all targets within 20 ft., DC 18 save or take 4d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, deadly",
			Tags: []string{"Thunderstone-Mine", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 18,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger20",
			Name: "Scythe Blade: DC 15 to find, DC 10 to disable; +12 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Tags: []string{"Scythe-Blade", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 19,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger21",
			Name: "Keelte: Female Elf Monk, LG. Str 12, Dex 14, Con 13, Int 9, Wis 15, Cha 14",
			Tags: []string{"Female", "DND-generator", "Elf", "Monk", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 20,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger22",
			Name: "Kather Larke: Female Halfling Cleric, CN. Str 8, Dex 8, Con 13, Int 7, Wis 13, Cha 10",
			Tags: []string{"Female", "DND-generator", "Halfling", "Cleric", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 21,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger23",
			Name: "Cyne: Male Human Soldier, NG. Str 12, Dex 9, Con 8, Int 10, Wis 8, Cha 10",
			Tags: []string{"Male", "DND-generator", "Human", "Soldier", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 22,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger24",
			Name: "Gytha: Female Human Barbarian, N. Str 16, Dex 13, Con 12, Int 12, Wis 14, Cha 9",
			Tags: []string{"Female", "DND-generator", "Human", "Barbarian", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 23,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger25",
			Name: "Brobern Hawte: Male Half-elf Monk, N. Str 12, Dex 10, Con 8, Int 14, Wis 12, Cha 12",
			Tags: []string{"Male", "DND-generator", "Half-elf", "Monk", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 24,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger26",
			Name: "Borneli: Male Elf Servant, LN. Str 12, Dex 12, Con 8, Int 13, Wis 6, Cha 12",
			Tags: []string{"Male", "DND-generator", "Elf", "Servant", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 25,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger27",
			Name: "Midda: Male Elf Sorcerer, LN. Str 10, Dex 13, Con 11, Int 7, Wis 10, Cha 13",
			Tags: []string{"Male", "DND-generator", "Elf", "Sorcerer", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 26,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger28",
			Name: "Burgwe: Female Human Bard, CN. Str 13, Dex 11, Con 10, Int 13, Wis 12, Cha 17.",
			Tags: []string{"Female", "DND-generator", "Human", "Bard", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 27,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger29",
			Name: "Carel: Female Gnome Druid, Neutral. Str 11, Dex 12, Con 7, Int 10, Wis 17, Cha 10",
			Tags: []string{"Female", "DND-generator", "Gnome", "Druid", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 28,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger30",
			Name: "Suse Salte: Female Human Aristocrat, N. Str 10, Dex 7, Con 10, Int 9, Wis 7, Cha 13",
			Tags: []string{"Female", "DND-generator", "Human", "Aristocrat", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 29,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger31",
			Name: "Surprise!",
			Tags: []string{"Something-extremely-new"},
		},
		LastCheck: moira.CheckData{
			Score: 30,
		},
	},
}
