package controller

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/dto"
	"github.com/moira-alert/moira/database"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateTrigger(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Success with trigger.ID empty", t, func() {
		triggerModel := dto.TriggerModel{}
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 15)
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
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 15)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), triggerModel.ToMoiraTrigger()).Return(nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger created")
		So(resp.ID, ShouldResemble, triggerID)
	})

	Convey("Success with custom valid trigger", t, func() {
		triggerID := "Valid.Custom_Trigger~Name-42"
		triggerModel := dto.TriggerModel{ID: triggerID}
		dataBase.EXPECT().GetTrigger(triggerModel.ID).Return(moira.Trigger{}, database.ErrNil)
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 15)
		dataBase.EXPECT().DeleteTriggerCheckLock(gomock.Any())
		dataBase.EXPECT().GetTriggerLastCheck(gomock.Any()).Return(moira.CheckData{}, database.ErrNil)
		dataBase.EXPECT().SetTriggerLastCheck(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
		dataBase.EXPECT().SaveTrigger(gomock.Any(), triggerModel.ToMoiraTrigger()).Return(nil)
		resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
		So(err, ShouldBeNil)
		So(resp.Message, ShouldResemble, "trigger created")
		So(resp.ID, ShouldResemble, triggerID)
	})

	Convey("Error with invalid triggerID", t, func() {
		for _, triggerID := range []string{"Foo#", "Foo%", "Foo^", "Foo ", "[Foo]", "Foo?", "Foo:@"} {
			triggerModel := dto.TriggerModel{ID: triggerID}
			resp, err := CreateTrigger(dataBase, &triggerModel, make(map[string]bool))
			expected := api.ErrorInvalidRequest(fmt.Errorf("trigger ID contains invalid characters (allowed: 0-9, a-z, A-Z, -, ~, _, .)"))
			So(err, ShouldResemble, expected)
			So(resp, ShouldBeNil)
		}
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
		dataBase.EXPECT().AcquireTriggerCheckLock(gomock.Any(), 15)
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

	Convey("Has triggers with metrics which DeletedButKept is true", t, func() {
		triggerIDs := []string{"1", "2", "3"}
		triggers := []*moira.TriggerCheck{
			{
				Throttling: 1,
				LastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {
							DeletedButKept: true,
						},
					},
				},
			},
			{
				Throttling: 2,
				LastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test2": {
							DeletedButKept: false,
						},
					},
				},
			},
			{
				Throttling: 3,
				LastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test3": {
							DeletedButKept: true,
						},
					},
				},
			},
		}

		expected := &dto.TriggersList{
			List: []moira.TriggerCheck{
				{
					Throttling: 1,
					LastCheck: moira.CheckData{
						Metrics: make(map[string]moira.MetricState),
					},
				},
				{
					Throttling: 2,
					LastCheck: moira.CheckData{
						Metrics: map[string]moira.MetricState{
							"test2": {
								DeletedButKept: false,
							},
						},
					},
				},
				{
					Throttling: 3,
					LastCheck: moira.CheckData{
						Metrics: make(map[string]moira.MetricState),
					},
				},
			},
		}

		mockDatabase.EXPECT().GetAllTriggerIDs().Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggers, nil)

		actual, err := GetAllTriggers(mockDatabase)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})
}

func TestSearchTriggers(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)
	mockIndex := mock_moira_alert.NewMockSearcher(mockCtrl)

	var exp int64 = 31
	testHighlights := make([]moira.SearchHighlight, 0)
	for field, value := range testHighlightsMap {
		testHighlights = append(testHighlights, moira.SearchHighlight{
			Field: field,
			Value: value,
		})
	}

	triggerSearchResults := make([]*moira.SearchResult, 0)
	for _, triggerCheck := range triggerChecks {
		triggerSearchResults = append(triggerSearchResults, &moira.SearchResult{
			ObjectID:   triggerCheck.ID,
			Highlights: testHighlights,
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

	searchOptions := moira.SearchOptions{
		Page:                  0,
		Size:                  50,
		OnlyProblems:          false,
		Tags:                  make([]string, 0),
		SearchString:          "",
		CreatedBy:             "",
		NeedSearchByCreatedBy: false,
		CreatePager:           false,
		PagerID:               "",
	}

	Convey("No tags, no text, onlyErrors = false, ", t, func() {
		Convey("With triggers which have metrics on Maintenance", func() {
			triggers := []*moira.TriggerCheck{
				{
					Throttling: 1,
					LastCheck: moira.CheckData{
						Metrics: map[string]moira.MetricState{
							"test1": {
								DeletedButKept: true,
							},
						},
					},
				},
				{
					Throttling: 2,
					LastCheck: moira.CheckData{
						Metrics: map[string]moira.MetricState{
							"test2": {},
						},
					},
				},
				{
					Throttling: 3,
					LastCheck: moira.CheckData{
						Metrics: map[string]moira.MetricState{
							"test3": {
								DeletedButKept: true,
							},
						},
					},
				},
			}

			mockIndex.EXPECT().SearchTriggers(searchOptions).Return([]*moira.SearchResult{
				{
					ObjectID: "1",
				},
				{
					ObjectID: "2",
				},
				{
					ObjectID: "3",
				},
			}, exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks([]string{"1", "2", "3"}).Return(triggers, nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List: []moira.TriggerCheck{
					{
						Throttling: 1,
						LastCheck: moira.CheckData{
							Metrics: map[string]moira.MetricState{},
						},
						Highlights: map[string]string{},
					},
					{
						Throttling: 2,
						LastCheck: moira.CheckData{
							Metrics: map[string]moira.MetricState{
								"test2": {},
							},
						},
						Highlights: map[string]string{},
					},
					{
						Throttling: 3,
						LastCheck: moira.CheckData{
							Metrics: map[string]moira.MetricState{},
						},
						Highlights: map[string]string{},
					},
				},
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
			})
		})

		Convey("Page is bigger than triggers number", func() {
			mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults, exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks,
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
			})
		})

		Convey("Must return all triggers, when size is -1", func() {
			searchOptions.Size = -1
			mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults, exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks,
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
			})
		})

		Convey("Page is less than triggers number", func() {
			searchOptions.Size = 10
			mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[:10], exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[:10]).Return(triggersPointers[:10], nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks[:10],
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
			})

			Convey("Second page", func() {
				searchOptions.Page = 1
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[10:20], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[10:20]).Return(triggersPointers[10:20], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[10:20],
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})
		})
	})

	Convey("Complex search query", t, func() {
		searchOptions.Size = 10
		searchOptions.Page = 0
		Convey("Only errors", func() {
			exp = 30
			// superTrigger31 is the only trigger without errors
			mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[:10], exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[:10]).Return(triggersPointers[:10], nil)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks[0:10],
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
			})

			Convey("Only errors with tags", func() {
				searchOptions.Tags = []string{"encounters", "Kobold"}
				exp = 2
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[1:3], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[1:3]).Return(triggersPointers[1:3], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[1:3],
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})

			Convey("Only errors with text terms", func() {
				searchOptions.SearchString = "dragonshield medium"
				exp = 1
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[2:3], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[2:3]).Return(triggersPointers[2:3], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[2:3],
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})

			Convey("Only errors with tags and text terms", func() {
				searchOptions.Tags = []string{"traps"}
				searchOptions.SearchString = "deadly"
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
						Highlights: testHighlights,
					})
					deadlyTrapsTriggerIDs = append(deadlyTrapsTriggerIDs, deadlyTrap.ID)
				}

				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(deadlyTrapsSearchResults, exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(deadlyTrapsTriggerIDs).Return(deadlyTrapsPointers, nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  deadlyTraps,
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})

			Convey("Only errors with createdBy", func() {
				searchOptions.CreatedBy = "monster"
				searchOptions.NeedSearchByCreatedBy = true
				exp = 7
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[9:16], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[9:16]).Return(triggersPointers[9:16], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[9:16],
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})

			Convey("Only errors with createdBy and tags", func() {
				searchOptions.CreatedBy = "tarasov.da"
				searchOptions.NeedSearchByCreatedBy = true
				searchOptions.Tags = []string{"Human", "NPCs"}
				exp = 2
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[22:24], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[22:24]).Return(triggersPointers[22:24], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[22:24],
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})

			Convey("Only errors with createdBy, tags and text terms", func() {
				searchOptions.CreatedBy = "internship2023"
				searchOptions.NeedSearchByCreatedBy = true
				searchOptions.Tags = []string{"Female", "NPCs"}
				searchOptions.SearchString = "Music"
				exp = 2
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[27:29], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[27:29]).Return(triggersPointers[27:29], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[27:29],
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})

			Convey("Only errors with EMPTY createdBy", func() {
				searchOptions.CreatedBy = ""
				searchOptions.NeedSearchByCreatedBy = true
				searchOptions.Tags = []string{}
				searchOptions.SearchString = ""
				exp = 3
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults[6:9], exp, nil)
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[6:9]).Return(triggersPointers[6:9], nil)
				list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
				So(err, ShouldBeNil)
				So(list, ShouldResemble, &dto.TriggersList{
					List:  triggerChecks[6:9],
					Total: &exp,
					Page:  &searchOptions.Page,
					Size:  &searchOptions.Size,
				})
			})
		})
	})

	Convey("Find triggers errors", t, func() {
		searchOptions = moira.SearchOptions{
			Page:                  0,
			Size:                  50,
			OnlyProblems:          false,
			Tags:                  make([]string, 0),
			SearchString:          "",
			CreatedBy:             "",
			NeedSearchByCreatedBy: false,
		}

		Convey("Error from searcher", func() {
			searcherError := fmt.Errorf("very bad request")
			mockIndex.EXPECT().SearchTriggers(searchOptions).Return(make([]*moira.SearchResult, 0), int64(0), searcherError)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldNotBeNil)
			So(list, ShouldBeNil)
		})

		Convey("Error from database", func() {
			searchOptions.Size = 50
			searcherError := fmt.Errorf("very bad request")
			mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults, exp, nil)
			mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(nil, searcherError)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldNotBeNil)
			So(list, ShouldBeNil)
		})

		Convey("Error on passed search elements and pagerID", func() {
			searchOptions.Tags = []string{"test"}
			searchOptions.SearchString = "test"
			searchOptions.PagerID = "test"
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldNotBeNil)
			So(list, ShouldBeNil)
		})
	})

	searchOptions.PagerID = ""

	Convey("Search with pager", t, func() {
		searchOptions.SearchString = ""
		searchOptions.Tags = []string{}
		Convey("Create pager", func() {
			searchOptions.Page = 0
			searchOptions.Size = -1
			searchOptions.CreatePager = true
			exp = 31
			gomock.InOrder(
				mockIndex.EXPECT().SearchTriggers(searchOptions).Return(triggerSearchResults, exp, nil),
				mockDatabase.EXPECT().SaveTriggersSearchResults(gomock.Any(), triggerSearchResults).Return(nil).Do(func(pID string, _ interface{}) {
					searchOptions.PagerID = pID
				}),
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil),
			)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks,
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
				Pager: &searchOptions.PagerID,
			})
		})

		Convey("Use pager", func() {
			searchOptions.PagerID = "TestPagerID"
			searchOptions.Page = 0
			searchOptions.Size = -1
			searchOptions.CreatePager = false
			var exp int64 = 31
			gomock.InOrder(
				mockDatabase.EXPECT().GetTriggersSearchResults(searchOptions.PagerID, searchOptions.Page, searchOptions.Size).Return(triggerSearchResults, exp, nil),
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggersPointers, nil),
			)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks,
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
				Pager: &searchOptions.PagerID,
			})
		})

		Convey("Use pager and page size higher than amount of search results", func() {
			searchOptions.PagerID = "TestPagerID"
			var exp int64 = 2
			searchOptions.Size = 10
			searchOptions.CreatePager = false
			gomock.InOrder(
				mockDatabase.EXPECT().GetTriggersSearchResults(searchOptions.PagerID, searchOptions.Page, searchOptions.Size).Return(triggerSearchResults[:2], exp, nil),
				mockDatabase.EXPECT().GetTriggerChecks(triggerIDs[:2]).Return(triggersPointers[:2], nil),
			)
			list, err := SearchTriggers(mockDatabase, mockIndex, searchOptions)
			So(err, ShouldBeNil)
			So(list, ShouldResemble, &dto.TriggersList{
				List:  triggerChecks[:2],
				Total: &exp,
				Page:  &searchOptions.Page,
				Size:  &searchOptions.Size,
				Pager: &searchOptions.PagerID,
			})
		})
	})
}

var triggerChecks = []moira.TriggerCheck{
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger1",
			Name:      "I used D&D character generator for test triggers: https://donjon.bin.sh",
			Tags:      []string{"DND-generator", "common"},
			CreatedBy: "test",
		},
		LastCheck: moira.CheckData{
			Score: 30,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger2",
			Name:      "Kobold Scale Sorcerer (cr 1, vgm 167) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Tags:      []string{"DND-generator", "Kobold", "Sorcerer", "encounters"},
			CreatedBy: "test",
		},
		LastCheck: moira.CheckData{
			Score: 29,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger3",
			Name:      "Kobold Dragonshield (cr 1, vgm 165) and 1 x Kobold (cr 1/8, mm 195); medium, 225 xp",
			Tags:      []string{"DND-generator", "Kobold", "Dragonshield", "encounters"},
			CreatedBy: "test",
		},
		LastCheck: moira.CheckData{
			Score: 28,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger4",
			Name:      "Orc Nurtured One of Yurtrus (cr 1/2, vgm 184) and 1 x Orc (cr 1/2, mm 246); hard, 200 xp",
			Tags:      []string{"DND-generator", "Orc", "encounters"},
			CreatedBy: "test",
		},
		LastCheck: moira.CheckData{
			Score: 27,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger5",
			Name:      "Rust Monster (cr 1/2, mm 262); easy, 100 xp",
			Tags:      []string{"DND-generator", "Rust-Monster", "encounters"},
			CreatedBy: "test",
		},
		LastCheck: moira.CheckData{
			Score: 26,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger6",
			Name:      "Giant Constrictor Snake (cr 2, mm 324); deadly, 450 xp",
			Tags:      []string{"Giant", "DND-generator", "Snake", "encounters"},
			CreatedBy: "test",
		},
		LastCheck: moira.CheckData{
			Score: 25,
		},
		Highlights: testHighlightsMap,
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
		Highlights: testHighlightsMap,
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
		Highlights: testHighlightsMap,
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
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger10",
			Name:      "Gibbering Mouther (cr 2, mm 157); easy, 450 xp",
			Tags:      []string{"Gibbering-Mouther", "DND-generator", "encounters"},
			CreatedBy: "monster",
		},
		LastCheck: moira.CheckData{
			Score: 21,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger11",
			Name:      "Scythe Blade: DC 10 to find, DC 10 to disable; +11 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Tags:      []string{"Scythe Blade", "DND-generator", "traps"},
			CreatedBy: "monster",
		},
		LastCheck: moira.CheckData{
			Score: 20,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger12",
			Name:      "Falling Block: DC 10 to find, DC 10 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags:      []string{"Falling-Block", "DND-generator", "traps"},
			CreatedBy: "monster",
		},
		LastCheck: moira.CheckData{
			Score: 19,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger13",
			Name:      "Thunderstone Mine: DC 15 to find, DC 15 to disable; affects all targets within 20 ft., DC 15 save or take 2d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, dangerous",
			Tags:      []string{"Thunderstone-Mine", "DND-generator", "traps"},
			CreatedBy: "monster",
		},
		LastCheck: moira.CheckData{
			Score: 18,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger14",
			Name:      "Falling Block: DC 10 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags:      []string{"Falling-Block", "DND-generator", "traps"},
			CreatedBy: "monster",
		},
		LastCheck: moira.CheckData{
			Score: 17,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger15",
			Name:      "Chain Flail: DC 15 to find, DC 10 to disable; initiative +3, 1 attack per round, +11 to hit against all targets within 5 ft., 4d10 bludgeoning damage; apprentice tier, deadly",
			Tags:      []string{"Chain-Flail", "DND-generator", "traps"},
			CreatedBy: "monster",
		},
		LastCheck: moira.CheckData{
			Score: 16,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger16",
			Name:      "Falling Block: DC 15 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 12 save or take 2d10 damage; apprentice tier, dangerous",
			Tags:      []string{"Falling-Block", "DND-generator", "traps"},
			CreatedBy: "monster",
		},
		LastCheck: moira.CheckData{
			Score: 15,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger17",
			Name:      "Electrified Floortile: DC 20 to find, DC 15 to disable; affects all targets within a 10 ft. square area, DC 15 save or take 2d10 lightning damage; apprentice tier, dangerous",
			Tags:      []string{"Electrified-Floortile", "DND-generator", "traps"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 14,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger18",
			Name:      "Earthmaw Trap: DC 15 to find, DC 10 to disable; +7 to hit against one target, 2d10 piercing damage; apprentice tier, dangerous",
			Tags:      []string{"Earthmaw-Trap", "DND-generator", "traps"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 13,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger19",
			Name:      "Thunderstone Mine: DC 15 to find, DC 20 to disable; affects all targets within 20 ft., DC 18 save or take 4d10 thunder damage and become deafened for 1d4 rounds; apprentice tier, deadly",
			Tags:      []string{"Thunderstone-Mine", "DND-generator", "traps"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 12,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger20",
			Name:      "Scythe Blade: DC 15 to find, DC 10 to disable; +12 to hit against all targets within a 5 ft. arc, 4d10 slashing damage; apprentice tier, deadly",
			Tags:      []string{"Scythe-Blade", "DND-generator", "traps"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 11,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger21",
			Name:      "Keelte: Female Elf Monk, LG. Str 12, Dex 14, Con 13, Int 9, Wis 15, Cha 14",
			Tags:      []string{"Female", "DND-generator", "Elf", "Monk", "NPCs"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 10,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger22",
			Name:      "Kather Larke: Female Halfling Cleric, CN. Str 8, Dex 8, Con 13, Int 7, Wis 13, Cha 10",
			Tags:      []string{"Female", "DND-generator", "Halfling", "Cleric", "NPCs"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 9,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger23",
			Name:      "Cyne: Male Human Soldier, NG. Str 12, Dex 9, Con 8, Int 10, Wis 8, Cha 10",
			Tags:      []string{"Male", "DND-generator", "Human", "Soldier", "NPCs"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 8,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger24",
			Name:      "Gytha: Female Human Barbarian, N. Str 16, Dex 13, Con 12, Int 12, Wis 14, Cha 9",
			Tags:      []string{"Female", "DND-generator", "Human", "Barbarian", "NPCs"},
			CreatedBy: "tarasov.da",
		},
		LastCheck: moira.CheckData{
			Score: 7,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger25",
			Name:      "Brobern Hawte: Male Half-elf Monk, N. Str 12, Dex 10, Con 8, Int 14, Wis 12, Cha 12",
			Tags:      []string{"Male", "DND-generator", "Half-elf", "Monk", "NPCs"},
			CreatedBy: "internship2023",
		},
		LastCheck: moira.CheckData{
			Score: 6,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger26",
			Name:      "Borneli: Male Elf Servant, LN. Str 12, Dex 12, Con 8, Int 13, Wis 6, Cha 12",
			Tags:      []string{"Male", "DND-generator", "Elf", "Servant", "NPCs"},
			CreatedBy: "internship2023",
		},
		LastCheck: moira.CheckData{
			Score: 5,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger27",
			Name:      "Midda: Male Elf Sorcerer, LN. Str 10, Dex 13, Con 11, Int 7, Wis 10, Cha 13",
			Tags:      []string{"Male", "DND-generator", "Elf", "Sorcerer", "NPCs"},
			CreatedBy: "internship2023",
		},
		LastCheck: moira.CheckData{
			Score: 4,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger28",
			Name:      "Burgwe: Female Human Bard, CN. Str 13, Dex 11, Con 10, Int 13, Wis 12, Cha 17. Music!",
			Tags:      []string{"Female", "DND-generator", "Human", "Bard", "NPCs"},
			CreatedBy: "internship2023",
		},
		LastCheck: moira.CheckData{
			Score: 3,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger29",
			Name:      "Carel: Female Gnome Druid, Neutral. Str 11, Dex 12, Con 7, Int 10, Wis 17, Cha 10. Music!",
			Tags:      []string{"Female", "DND-generator", "Gnome", "Druid", "NPCs"},
			CreatedBy: "internship2023",
		},
		LastCheck: moira.CheckData{
			Score: 2,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger30",
			Name:      "Suse Salte: Female Human Aristocrat, N. Str 10, Dex 7, Con 10, Int 9, Wis 7, Cha 13",
			Tags:      []string{"Female", "DND-generator", "Human", "Aristocrat", "NPCs"},
			CreatedBy: "internship2023",
		},
		LastCheck: moira.CheckData{
			Score: 1,
		},
		Highlights: testHighlightsMap,
	},
	{
		Trigger: moira.Trigger{
			ID:        "SuperTrigger31",
			Name:      "Surprise!",
			Tags:      []string{"Something-extremely-new"},
			CreatedBy: "internship2023",
		},
		LastCheck: moira.CheckData{
			Score: 0,
		},
		Highlights: testHighlightsMap,
	},
}

var testHighlightsMap = map[string]string{"testField": "testHighlight"}

func TestDeleteTriggersPager(t *testing.T) {
	Convey("DeleteTriggersPager", t, func() {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()
		dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)
		const pagerID = "pagerID"

		Convey("Pager exists", func() {
			dataBase.EXPECT().IsTriggersSearchResultsExist(pagerID).Return(true, nil)
			dataBase.EXPECT().DeleteTriggersSearchResults(pagerID).Return(nil)
			response, err := DeleteTriggersPager(dataBase, pagerID)
			So(err, ShouldBeNil)
			So(response, ShouldResemble, dto.TriggersSearchResultDeleteResponse{PagerID: pagerID})
		})

		Convey("Pager is not exist", func() {
			dataBase.EXPECT().IsTriggersSearchResultsExist(pagerID).Return(false, nil)
			response, err := DeleteTriggersPager(dataBase, pagerID)
			So(err, ShouldResemble, api.ErrorNotFound("pager with id pagerID not found"))
			So(response, ShouldResemble, dto.TriggersSearchResultDeleteResponse{})
		})

		Convey("Error while checking pager existence", func() {
			errReturning := errors.New("example error")
			dataBase.EXPECT().IsTriggersSearchResultsExist(pagerID).Return(false, errReturning)
			response, err := DeleteTriggersPager(dataBase, pagerID)
			So(err, ShouldResemble, api.ErrorInternalServer(errReturning))
			So(response, ShouldResemble, dto.TriggersSearchResultDeleteResponse{})
		})

		Convey("Error while deleting pager", func() {
			errReturning := errors.New("example error")
			dataBase.EXPECT().IsTriggersSearchResultsExist(pagerID).Return(true, nil)
			dataBase.EXPECT().DeleteTriggersSearchResults(pagerID).Return(errReturning)
			response, err := DeleteTriggersPager(dataBase, pagerID)
			So(err, ShouldResemble, api.ErrorInternalServer(errReturning))
			So(response, ShouldResemble, dto.TriggersSearchResultDeleteResponse{})
		})
	})
}

func TestGetUnusedTriggerIDs(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabase := mock_moira_alert.NewMockDatabase(mockCtrl)

	Convey("Has triggers", t, func() {
		triggerIDs := []string{uuid.Must(uuid.NewV4()).String(), uuid.Must(uuid.NewV4()).String()}
		triggers := []*moira.TriggerCheck{{Trigger: moira.Trigger{ID: triggerIDs[0]}}, {Trigger: moira.Trigger{ID: triggerIDs[1]}}}
		triggersList := []moira.TriggerCheck{{Trigger: moira.Trigger{ID: triggerIDs[0]}}, {Trigger: moira.Trigger{ID: triggerIDs[1]}}}
		mockDatabase.EXPECT().GetUnusedTriggerIDs().Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggers, nil)
		list, err := GetUnusedTriggerIDs(mockDatabase)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: triggersList})
	})

	Convey("No triggers", t, func() {
		mockDatabase.EXPECT().GetUnusedTriggerIDs().Return(make([]string, 0), nil)
		mockDatabase.EXPECT().GetTriggerChecks(make([]string, 0)).Return(make([]*moira.TriggerCheck, 0), nil)
		list, err := GetUnusedTriggerIDs(mockDatabase)
		So(err, ShouldBeNil)
		So(list, ShouldResemble, &dto.TriggersList{List: make([]moira.TriggerCheck, 0)})
	})

	Convey("GetUnusedTriggerIDs error", t, func() {
		expected := fmt.Errorf("getTriggerIDs error")
		mockDatabase.EXPECT().GetUnusedTriggerIDs().Return(nil, expected)
		list, err := GetUnusedTriggerIDs(mockDatabase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("GetTriggerChecks error", t, func() {
		expected := fmt.Errorf("getTriggerChecks error")
		mockDatabase.EXPECT().GetUnusedTriggerIDs().Return(make([]string, 0), nil)
		mockDatabase.EXPECT().GetTriggerChecks(make([]string, 0)).Return(nil, expected)
		list, err := GetUnusedTriggerIDs(mockDatabase)
		So(err, ShouldResemble, api.ErrorInternalServer(expected))
		So(list, ShouldBeNil)
	})

	Convey("Has triggers with metrics which DeletedButKept is true", t, func() {
		triggerIDs := []string{"1", "2", "3"}
		triggers := []*moira.TriggerCheck{
			{
				Throttling: 1,
				LastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test1": {
							DeletedButKept: true,
						},
					},
				},
			},
			{
				Throttling: 2,
				LastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test2": {
							DeletedButKept: false,
						},
					},
				},
			},
			{
				Throttling: 3,
				LastCheck: moira.CheckData{
					Metrics: map[string]moira.MetricState{
						"test3": {
							DeletedButKept: true,
						},
					},
				},
			},
		}

		expected := &dto.TriggersList{
			List: []moira.TriggerCheck{
				{
					Throttling: 1,
					LastCheck: moira.CheckData{
						Metrics: make(map[string]moira.MetricState),
					},
				},
				{
					Throttling: 2,
					LastCheck: moira.CheckData{
						Metrics: map[string]moira.MetricState{
							"test2": {
								DeletedButKept: false,
							},
						},
					},
				},
				{
					Throttling: 3,
					LastCheck: moira.CheckData{
						Metrics: make(map[string]moira.MetricState),
					},
				},
			},
		}

		mockDatabase.EXPECT().GetUnusedTriggerIDs().Return(triggerIDs, nil)
		mockDatabase.EXPECT().GetTriggerChecks(triggerIDs).Return(triggers, nil)

		actual, err := GetUnusedTriggerIDs(mockDatabase)
		So(err, ShouldBeNil)
		So(actual, ShouldResemble, expected)
	})
}
