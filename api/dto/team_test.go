package dto

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/moira-alert/moira/api"
	"github.com/moira-alert/moira/api/middleware"

	. "github.com/smartystreets/goconvey/convey"
)

func TestTeamValidation(t *testing.T) {
	Convey("Test team validation", t, func() {
		teamModel := TeamModel{}

		limits := api.GetTestLimitsConfig()

		request, _ := http.NewRequest(http.MethodPost, "/api/teams", nil)
		request.Header.Set("Content-Type", "application/json")
		request = request.WithContext(middleware.SetContextValueForTest(request.Context(), "limits", limits))

		Convey("with empty team.Name", func() {
			err := teamModel.Bind(request)

			So(err, ShouldResemble, errEmptyTeamName)
		})

		Convey("with team.Name has characters more than in limit", func() {
			teamModel.Name = strings.Repeat("ё", limits.Team.MaxNameSize+1)

			err := teamModel.Bind(request)

			So(err, ShouldResemble, fmt.Errorf("team name cannot be longer than %d characters", limits.Team.MaxNameSize))
		})

		Convey("with team.Description has characters more than in limit", func() {
			teamModel.Name = strings.Repeat("ё", limits.Team.MaxNameSize)
			teamModel.Description = strings.Repeat("ё", limits.Team.MaxDescriptionSize+1)

			err := teamModel.Bind(request)

			So(err, ShouldResemble, fmt.Errorf("team description cannot be longer than %d characters", limits.Team.MaxDescriptionSize))
		})

		Convey("with valid team", func() {
			teamModel.Name = strings.Repeat("ё", limits.Team.MaxNameSize)
			teamModel.Description = strings.Repeat("ё", limits.Team.MaxDescriptionSize)

			err := teamModel.Bind(request)

			So(err, ShouldBeNil)
		})
	})
}
