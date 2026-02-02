package redis

import (
	"fmt"
	"testing"

	"github.com/moira-alert/moira"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
	mock_clock "github.com/moira-alert/moira/mock/clock"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSelfCheckWithWritesInChecker(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.source = Checker
	dataBase.Flush()

	defer dataBase.Flush()

	defaultLocalCluster := moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster)
	defaultRemoteCluster := moira.DefaultGraphiteRemoteCluster

	Convey("Self state triggers manipulation", t, func() {
		Convey("Empty config", func() {
			count, err := dataBase.GetMetricsUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)

			count, err = dataBase.GetChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})

		Convey("Update metrics heartbeat test", func() {
			err := dataBase.UpdateMetricsHeartbeat()
			So(err, ShouldBeNil)

			count, err := dataBase.GetMetricsUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})

		Convey("Update metrics checks updates count", func() {
			err := dataBase.SetTriggerLastCheck("123", &lastCheckTest, defaultLocalCluster)
			So(err, ShouldBeNil)

			count, err := dataBase.GetChecksUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)

			err = dataBase.SetTriggerLastCheck("12345", &lastCheckTest, defaultRemoteCluster)
			So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			So(count, ShouldEqual, 1)
			So(err, ShouldBeNil)
		})
	})
}

func TestSelfCheckWithWritesNotInChecker(t *testing.T) {
	dbSources := []DBSource{Filter, API, Notifier, Cli, testSource}
	for _, dbSource := range dbSources {
		testSelfCheckWithWritesInDBSource(t, dbSource)
	}
}

func testSelfCheckWithWritesInDBSource(t *testing.T, dbSource DBSource) {
	t.Helper()

	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	dataBase.source = dbSource
	dataBase.Flush()

	defer dataBase.Flush()

	defaultLocalCluster := moira.MakeClusterKey(moira.GraphiteLocal, moira.DefaultCluster)
	defaultRemoteCluster := moira.DefaultGraphiteRemoteCluster

	Convey(fmt.Sprintf("Self state triggers manipulation in %s", dbSource), t, func() {
		Convey("Update metrics checks updates count", func() {
			err := dataBase.SetTriggerLastCheck("123", &lastCheckTest, defaultLocalCluster)
			So(err, ShouldBeNil)

			count, err := dataBase.GetChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)

			err = dataBase.SetTriggerLastCheck("12345", &lastCheckTest, defaultRemoteCluster)
			So(err, ShouldBeNil)

			count, err = dataBase.GetRemoteChecksUpdatesCount()
			So(count, ShouldEqual, 0)
			So(err, ShouldBeNil)
		})
	})
}

func TestSelfCheckErrorConnection(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabaseWithIncorrectConfig(logger)
	dataBase.Flush()

	defer dataBase.Flush()

	Convey("Should throw error when no connection", t, func() {
		count, err := dataBase.GetMetricsUpdatesCount()
		So(count, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		count, err = dataBase.GetChecksUpdatesCount()
		So(count, ShouldEqual, 0)
		So(err, ShouldNotBeNil)

		err = dataBase.UpdateMetricsHeartbeat()
		So(err, ShouldNotBeNil)
	})
}

func TestNotifierState(t *testing.T) {
	mock := gomock.NewController(t)
	logger, _ := logging.GetLogger("dataBase")
	dataBase := NewTestDatabase(logger)
	emptyDataBase := NewTestDatabaseWithIncorrectConfig(logger)

	dataBase.Flush()

	defer dataBase.Flush()

	Convey(fmt.Sprintf("Test on empty key '%v'", selfStateNotifierHealth), t, func() {
		Convey("On empty database should return ERROR", func() {
			clock := mock_clock.NewMockClock(mock)
			clock.EXPECT().NowUnix().Return(int64(0)).Times(2)
			notifierState, err := emptyDataBase.GetNotifierState(clock)
			So(notifierState, ShouldResemble, moira.NotifierState{
				State: moira.SelfStateERROR,
				Actor: moira.SelfStateActorAutomatic,
			})
			So(err, ShouldNotBeNil)
		})
		Convey("On real database should return OK", func() {
			clock := mock_clock.NewMockClock(mock)
			clock.EXPECT().NowUnix().Return(int64(0))
			notifierState, err := dataBase.GetNotifierState(clock)
			So(notifierState, ShouldResemble, moira.NotifierState{
				State: moira.SelfStateOK,
				Actor: moira.SelfStateActorManual,
			})
			So(err, ShouldBeNil)
		})
	})

	Convey(fmt.Sprintf("Test setting '%v' and reading it back", selfStateNotifierHealth), t, func() {
		Convey("Switch notifier to OK", func() {
			clock := mock_clock.NewMockClock(mock)
			clock.EXPECT().NowUnix().Return(int64(0))
			err := dataBase.SetNotifierState(moira.SelfStateActorManual, moira.SelfStateOK, clock)
			actualNotifierState, err2 := dataBase.GetNotifierState(clock)

			So(actualNotifierState.State, ShouldEqual, moira.SelfStateOK)
			So(err, ShouldBeNil)
			So(err2, ShouldBeNil)
		})

		Convey("Switch notifier to ERROR", func() {
			clock := mock_clock.NewMockClock(mock)
			clock.EXPECT().NowUnix().Return(int64(0))
			err := dataBase.SetNotifierState(moira.SelfStateActorManual, moira.SelfStateERROR, clock)
			actualNotifierState, err2 := dataBase.GetNotifierState(clock)

			So(actualNotifierState.State, ShouldEqual, moira.SelfStateERROR)
			So(err, ShouldBeNil)
			So(err2, ShouldBeNil)
		})
	})
}

func TestSetNotifierStateForSource(t *testing.T) {
	mock := gomock.NewController(t)
	logger, _ := logging.GetLogger("dataBase")
	database := NewTestDatabase(logger)

	database.Flush()

	t.Run("Test set no states was set fot sources", func(t *testing.T) {
		defer database.Flush()

		clock := mock_clock.NewMockClock(mock)
		clock.EXPECT().NowUnix().Return(int64(0)).Times(3)

		state, err := database.GetNotifierStateForSources(clock)
		require.NoError(t, err)
		require.Equal(t, map[moira.ClusterKey]moira.NotifierState{
			moira.DefaultLocalCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
			moira.DefaultGraphiteRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
			moira.DefaultPrometheusRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
		}, state)
	})

	t.Run("Test set ERROR for a single source", func(t *testing.T) {
		defer database.Flush()

		clock := mock_clock.NewMockClock(mock)
		clock.EXPECT().NowUnix().Return(int64(0)).Times(3)

		err := database.SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorManual, moira.SelfStateERROR, clock)
		require.NoError(t, err)

		state, err := database.GetNotifierStateForSources(clock)
		require.NoError(t, err)
		require.Equal(t, map[moira.ClusterKey]moira.NotifierState{
			moira.DefaultLocalCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateERROR,
			},
			moira.DefaultGraphiteRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
			moira.DefaultPrometheusRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
		}, state)
	})

	t.Run("Test set ERROR for a several sources", func(t *testing.T) {
		defer database.Flush()

		clock := mock_clock.NewMockClock(mock)
		clock.EXPECT().NowUnix().Return(int64(0)).Times(3)

		err := database.SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorManual, moira.SelfStateERROR, clock)
		require.NoError(t, err)
		err = database.SetNotifierStateForSource(moira.DefaultGraphiteRemoteCluster, moira.SelfStateActorManual, moira.SelfStateERROR, clock)
		require.NoError(t, err)

		state, err := database.GetNotifierStateForSources(clock)
		require.NoError(t, err)
		require.Equal(t, map[moira.ClusterKey]moira.NotifierState{
			moira.DefaultLocalCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateERROR,
			},
			moira.DefaultGraphiteRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateERROR,
			},
			moira.DefaultPrometheusRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
		}, state)
	})

	t.Run("Set source state to ERROR and then to OK", func(t *testing.T) {
		defer database.Flush()

		clock := mock_clock.NewMockClock(mock)
		clock.EXPECT().NowUnix().Return(int64(0)).Times(4)

		err := database.SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorManual, moira.SelfStateERROR, clock)
		require.NoError(t, err)
		err = database.SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorManual, moira.SelfStateOK, clock)
		require.NoError(t, err)

		state, err := database.GetNotifierStateForSources(clock)
		require.NoError(t, err)
		require.Equal(t, map[moira.ClusterKey]moira.NotifierState{
			moira.DefaultLocalCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
			moira.DefaultGraphiteRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
			moira.DefaultPrometheusRemoteCluster: {
				Actor: moira.SelfStateActorManual,
				State: moira.SelfStateOK,
			},
		}, state)
	})

	t.Run("Set nonexisting source must return err", func(t *testing.T) {
		defer database.Flush()

		clock := mock_clock.NewMockClock(mock)

		clusterKey := moira.ClusterKey{
			TriggerSource: moira.PrometheusRemote,
			ClusterId:     moira.ClusterId("cluster_that_does_not_exist"),
		}

		err := database.SetNotifierStateForSource(clusterKey, moira.SelfStateActorManual, moira.SelfStateERROR, clock)
		require.Error(t, err, "unknown cluster 'prometheus_remote.cluster_that_does_not_exist'")
	})
}

func TestGetNotifierStateForSource(t *testing.T) {
	logger, _ := logging.GetLogger("dataBase")
	database := NewTestDatabase(logger)
	database.Flush()

	mock := gomock.NewController(t)

	clock := mock_clock.NewMockClock(mock)
	clock.EXPECT().NowUnix().Return(int64(0)).Times(3)

	defer database.Flush()

	t.Run("Get state for all sources one by one", func(t *testing.T) {
		err := database.SetNotifierStateForSource(moira.DefaultLocalCluster, moira.SelfStateActorManual, moira.SelfStateERROR, clock)
		require.NoError(t, err)

		state, err := database.GetNotifierStateForSource(moira.DefaultLocalCluster, clock)
		require.NoError(t, err)
		require.Equal(t, moira.NotifierState{
			Actor: moira.SelfStateActorManual,
			State: moira.SelfStateERROR,
		}, state)

		state, err = database.GetNotifierStateForSource(moira.DefaultGraphiteRemoteCluster, clock)
		require.NoError(t, err)
		require.Equal(t, moira.NotifierState{
			Actor: moira.SelfStateActorManual,
			State: moira.SelfStateOK,
		}, state)

		state, err = database.GetNotifierStateForSource(moira.DefaultPrometheusRemoteCluster, clock)
		require.NoError(t, err)
		require.Equal(t, moira.NotifierState{
			Actor: moira.SelfStateActorManual,
			State: moira.SelfStateOK,
		}, state)
	})
}
