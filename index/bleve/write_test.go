package bleve

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/index/mapping"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTriggerIndex_Write(t *testing.T) {
	var newIndex *TriggerIndex
	var err error
	var count int64

	triggerMapping := mapping.BuildIndexMapping(mapping.Trigger{})

	triggerIDs := make([]string, len(triggerChecks))
	for i, trigger := range triggerChecks {
		triggerIDs[i] = trigger.ID
	}

	triggersPointers := make([]*moira.TriggerCheck, len(triggerChecks))
	for i, trigger := range triggerChecks {
		newTrigger := new(moira.TriggerCheck)
		*newTrigger = trigger
		triggersPointers[i] = newTrigger
	}

	Convey("First of all, create index", t, func() {
		newIndex, err = CreateTriggerIndex(triggerMapping)
		So(newIndex, ShouldHaveSameTypeAs, &TriggerIndex{})
		So(err, ShouldBeNil)

		count, err = newIndex.GetCount()
		So(count, ShouldBeZeroValue)
		So(err, ShouldBeNil)
	})

	Convey("Test write triggers and get count", t, func() {

		Convey("Test write 0 triggers", func() {
			err = newIndex.Write(triggersPointers[0:0])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldBeZeroValue)
			So(err, ShouldBeNil)
		})

		Convey("Test write 1 trigger", func() {
			err = newIndex.Write(triggersPointers[0:1])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(1))
			So(err, ShouldBeNil)
		})

		Convey("Test write the same 1 trigger", func() {
			err = newIndex.Write(triggersPointers[0:1])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(1))
			So(err, ShouldBeNil)
		})

		Convey("Test write 10 triggers", func() {
			err = newIndex.Write(triggersPointers[0:10])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(10))
			So(err, ShouldBeNil)
		})

		Convey("Test write the same 10 triggers", func() {
			err = newIndex.Write(triggersPointers[0:10])
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(10))
			So(err, ShouldBeNil)
		})

		Convey("Test write all 31 triggers", func() {
			err = newIndex.Write(triggersPointers)
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(31))
			So(err, ShouldBeNil)
		})

		Convey("Test write the same 31 triggers", func() {
			err = newIndex.Write(triggersPointers)
			So(err, ShouldBeNil)

			count, err = newIndex.GetCount()
			So(count, ShouldEqual, int64(31))
			So(err, ShouldBeNil)
		})

	})
}

var triggerChecks = []moira.TriggerCheck{
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger1",
			Name: "I used D&D character generator for test triggers: https://donjon.bin.sh",
			Desc: &descriptions[0],
			Tags: []string{"DND-generator", "common"},
		},
		LastCheck: moira.CheckData{
			Score: 30,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger2",
			Name: "Kobold Scale Sorcerer (cr 1, vgm 167) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Desc: &descriptions[1],
			Tags: []string{"DND-generator", "Kobold", "Sorcerer", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 29,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger3",
			Name: "Kobold Dragonshield (cr 1, vgm 165) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Desc: &descriptions[2],
			Tags: []string{"DND-generator", "Kobold", "Dragonshield", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 28,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger4",
			Name: "Orc Nurtured One of Yurtrus (cr 1/2, vgm 184) and 1 x Orc (cr 1/2, mm 246); hard, 200 xp",
			Desc: &descriptions[3],
			Tags: []string{"DND-generator", "Orc", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 27,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger5",
			Name: "Rust Monster (cr 1/2, mm 262); easy, 100 xp",
			Desc: &descriptions[4],
			Tags: []string{"DND-generator", "Rust-Monster", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 26,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger6",
			Name: "Giant Constrictor Snake (cr 2, mm 324); deadly, 450 xp",
			Desc: &descriptions[5],
			Tags: []string{"Giant", "DND-generator", "Snake", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 25,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger7",
			Name: "Darkling (cr 1/2, vgm 134); hard, 200 xp",
			Desc: &descriptions[6],
			Tags: []string{"Darkling", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 24,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger8",
			Name: "Ghost (cr 4, mm 147); hard, 1100 xp",
			Desc: &descriptions[7],
			Tags: []string{"Ghost", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 23,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger9",
			Name: "Spectator (cr 3, mm 30); medium, 700 xp",
			Desc: &descriptions[8],
			Tags: []string{"Spectator", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 22,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger10",
			Name: "Gibbering Mouther (cr 2, mm 157); easy, 450 xp",
			Desc: &descriptions[9],
			Tags: []string{"Gibbering-Mouther", "DND-generator", "encounters"},
		},
		LastCheck: moira.CheckData{
			Score: 21,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger11",
			Name: "Scythe Blade: DC 10 to find, DC 10 to disable; +11 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Desc: &descriptions[10],
			Tags: []string{"Scythe Blade", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 20,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger12",
			Name: "Falling Block: DC 10 to find, DC 10 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Desc: &descriptions[11],
			Tags: []string{"Falling-Block", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 19,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger13",
			Name: "Thunderstone Mine: DC 15 to find, DC 15 to disable; affects all targets within 20 ft., DC 15 save or take 2d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, dangerous",
			Desc: &descriptions[12],
			Tags: []string{"Thunderstone-Mine", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 18,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger14",
			Name: "Falling Block: DC 10 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Desc: &descriptions[13],
			Tags: []string{"Falling-Block", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 17,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger15",
			Name: "Chain Flail: DC 15 to find, DC 10 to disable; initiative +3, 1 attack per round, +11 to hit against all targets within 5 ft., 4d10 bludgeoning damage; apprentice tier, deadly",
			Desc: &descriptions[14],
			Tags: []string{"Chain-Flail", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 16,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger16",
			Name: "Falling Block: DC 15 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Desc: &descriptions[15],
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
			Desc: &descriptions[16],
			Tags: []string{"Electrified-Floortile", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 14,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger18",
			Name: "Earthmaw Trap: DC 15 to find, DC 10 to disable; +7 to hit against one target, 2d10 piercing damage; apprentice tier, dangerous",
			Desc: &descriptions[17],
			Tags: []string{"Earthmaw-Trap", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 13,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger19",
			Name: "Thunderstone Mine: DC 15 to find, DC 20 to disable; affects all targets within 20 ft., DC 18 save or take 4d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, deadly",
			Desc: &descriptions[18],
			Tags: []string{"Thunderstone-Mine", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 12,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger20",
			Name: "Scythe Blade: DC 15 to find, DC 10 to disable; +12 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Desc: &descriptions[19],
			Tags: []string{"Scythe-Blade", "DND-generator", "traps"},
		},
		LastCheck: moira.CheckData{
			Score: 11,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger21",
			Name: "Keelte: Female Elf Monk, LG. Str 12, Dex 14, Con 13, Int 9, Wis 15, Cha 14",
			Desc: &descriptions[20],
			Tags: []string{"Female", "DND-generator", "Elf", "Monk", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 10,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger22",
			Name: "Kather Larke: Female Halfling Cleric, CN. Str 8, Dex 8, Con 13, Int 7, Wis 13, Cha 10",
			Desc: &descriptions[21],
			Tags: []string{"Female", "DND-generator", "Halfling", "Cleric", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 9,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger23",
			Name: "Cyne: Male Human Soldier, NG. Str 12, Dex 9, Con 8, Int 10, Wis 8, Cha 10",
			Desc: &descriptions[22],
			Tags: []string{"Male", "DND-generator", "Human", "Soldier", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 8,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger24",
			Name: "Gytha: Female Human Barbarian, N. Str 16, Dex 13, Con 12, Int 12, Wis 14, Cha 9",
			Desc: &descriptions[23],
			Tags: []string{"Female", "DND-generator", "Human", "Barbarian", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 7,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger25",
			Name: "Brobern Hawte: Male Half-elf Monk, N. Str 12, Dex 10, Con 8, Int 14, Wis 12, Cha 12",
			Desc: &descriptions[24],
			Tags: []string{"Male", "DND-generator", "Half-elf", "Monk", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 6,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger26",
			Name: "Borneli: Male Elf Servant, LN. Str 12, Dex 12, Con 8, Int 13, Wis 6, Cha 12",
			Desc: &descriptions[25],
			Tags: []string{"Male", "DND-generator", "Elf", "Servant", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 5,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger27",
			Name: "Midda: Male Elf Sorcerer, LN. Str 10, Dex 13, Con 11, Int 7, Wis 10, Cha 13",
			Desc: &descriptions[26],
			Tags: []string{"Male", "DND-generator", "Elf", "Sorcerer", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 4,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger28",
			Name: "Burgwe: Female Human Bard, CN. Str 13, Dex 11, Con 10, Int 13, Wis 12, Cha 17.",
			Desc: &descriptions[27],
			Tags: []string{"Female", "DND-generator", "Human", "Bard", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 3,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger29",
			Name: "Carel: Female Gnome Druid, Neutral. Str 11, Dex 12, Con 7, Int 10, Wis 17, Cha 10",
			Desc: &descriptions[28],
			Tags: []string{"Female", "DND-generator", "Gnome", "Druid", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 2,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger30",
			Name: "Suse Salte: Female Human Aristocrat, N. Str 10, Dex 7, Con 10, Int 9, Wis 7, Cha 13",
			Desc: &descriptions[29],
			Tags: []string{"Female", "DND-generator", "Human", "Aristocrat", "NPCs"},
		},
		LastCheck: moira.CheckData{
			Score: 1,
		},
	},
	{
		Trigger: moira.Trigger{
			ID:   "SuperTrigger31",
			Name: "Surprise!",
			Desc: &descriptions[30],
			Tags: []string{"Something-extremely-new"},
		},
		LastCheck: moira.CheckData{
			Score: 0,
		},
	},
}

var descriptions = []string{
	"0: Is this the real life? Is this just fantasy?",
	"1: Caught in a landslide, no escape from reality",
	"2: Open your eyes, look up to the skies and see",
	"3: I'm just a poor boy, I need no sympathy",
	"4: Because I'm easy come, easy go, little high, little low",
	"5: Any way the wind blows doesn't really matter to me, to me",
	"6",
	"7: Mama, just killed a man",
	"8: Put a gun against his head, pulled my trigger, now he's dead",
	"9: Mama, life had just begun",
	"10: But now I've gone and thrown it all away",
	"11: Mama, ooh, didn't mean to make you cry",
	"12: If I'm not back again this time tomorrow",
	"13: Carry on, carry on as if nothing really matters",
	"14",
	"15: Too late, my time has come",
	"16: Sends shivers down my spine, body's aching all the time",
	"17: Goodbye, everybody, I've got to go",
	"18: Gotta leave you all behind and face the truth",
	"19: Mama, ooh, (Any way the wind blows)",
	"20: I don't wanna die",
	"21: I sometimes wish I'd never been born at all",
	"22",
	"23: I see a little silhouetto of a man",
	"24: Scaramouche, Scaramouche, will you do the Fandango?",
	"25: Thunderbolt and lightning, very, very fright'ning me",
	"26: (Galileo) Galileo, (Galileo) Galileo, Galileo Figaro magnifico",
	"27: I'm just a poor boy, nobody loves me",
	"28: He's just a poor boy from a poor family",
	"29: Spare him his life from this monstrosity",
	`30: Easy come, easy go, will you let me go?
Bismillah! No, we will not let you go
(Let him go!) Bismillah! We will not let you go
(Let him go!) Bismillah! We will not let you go
(Let me go) Will not let you go
(Let me go) Will not let you go
(Let me go) Ah
No, no, no, no, no, no, no
(Oh mamma mia, mamma mia) Mamma mia, let me go
Beelzebub has a devil put aside for me, for me, for me!`,
}
