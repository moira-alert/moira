package controller

import (
	"testing"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/api/dto"
	mock_moira_alert "github.com/moira-alert/moira/mock/moira-alert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	t.Run("On startup should return OK", func(t *testing.T) {
		expectedState := dto.NotifierState{State: moira.SelfStateOK}
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{State: moira.SelfStateOK}, nil)

		actualState, err := GetNotifierState(dataBase)

		require.Nil(t, err)
		assert.Equal(t, expectedState, *actualState)
	})
}

func TestUpdateNotifierState(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	dataBase := mock_moira_alert.NewMockDatabase(mockCtrl)

	t.Run("Setting OK notifier state", func(t *testing.T) {
		expectedState := dto.NotifierState{State: moira.SelfStateOK}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateActorManual, moira.SelfStateOK).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{State: moira.SelfStateOK}, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateOK})
		require.Nil(t, err)

		actualState, err := GetNotifierState(dataBase)
		require.Nil(t, err)
		assert.Equal(t, expectedState, *actualState)
	})

	t.Run("Setting ERROR notifier state", func(t *testing.T) {
		expectedState := dto.NotifierState{State: moira.SelfStateERROR, Message: dto.ErrorMessage}
		dataBase.EXPECT().SetNotifierState(moira.SelfStateActorManual, moira.SelfStateERROR).Return(nil)
		dataBase.EXPECT().GetNotifierState().Return(moira.NotifierState{State: moira.SelfStateERROR}, nil)

		err := UpdateNotifierState(dataBase, &dto.NotifierState{State: moira.SelfStateERROR})
		require.Nil(t, err)

		actualState, err := GetNotifierState(dataBase)
		require.Nil(t, err)
		assert.Equal(t, expectedState, *actualState)
	})
}
