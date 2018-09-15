package matched

import (
	"fmt"
	"math"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/op/go-logging"

	"github.com/moira-alert/moira/mock/moira-alert"
	"github.com/FZambia/go-sentinel"
)

type Sample struct {
	Name       string
	Serie      []float64
	StopPoints []float64
}

var samples = []Sample{
	{
		Name: "520",
		Serie: []float64{
			1408.1, 1385.8666666666666, 1397.2666666666667, 1382.2833333333333, 1408.9333333333334,
			math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
			math.NaN(), math.NaN(), math.NaN(), 5685.466666666666, 6054.966666666666, 3390.116666666667,
			1389.4, 1375.4, 1393.65, 1383.0166666666667, 1404.2333333333333, 1390.5666666666666, 1389.25,
			1394.0833333333333, 1404.8166666666666, 1385.9666666666667, 1387.6333333333332, 1379.15,
			math.NaN(), 1380.35, 1375.35, 1396.6666666666667, 1374.0166666666667, 1402.7833333333333,
			1373.5666666666666, 1399.4333333333334, 1375.6166666666666, 1393.1166666666666,
			1393.75, 1390.2166666666667, 1399.1333333333332, 1388.2333333333333, 1391.0833333333333,
			1371.5333333333333, 1395.3, 1381.4666666666667, 1399.0333333333333, 1391.7,
		},
		StopPoints: []float64{
			1408.9333333333334,
		},
	},
	{
		Name: "616.1",
		Serie: []float64{
			1290.6333333333332, math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
			2072.6, 1402.1333333333332, 1311.1833333333334, 1296.8833333333332, 1313.4, 1306.1833333333334,
			1451.0666666666666, 1290.5, 1308.5833333333333, 1311.45, 1311.8333333333333, 1297.0166666666667,
			1321.1333333333332, 1312, 1309.9666666666667, 1328.3666666666666, 1298.5333333333333,
			1310.9166666666667, 1303.7,
		},
		StopPoints: []float64{
			1290.6333333333332,
		},
	},
	{
		Name: "616.2",
		Serie: []float64{
			1234.5833333333333, math.NaN(), math.NaN(), math.NaN(), math.NaN(), math.NaN(),
			2014.0833333333333, 1337.9166666666667, 1258.3, 1234.65, 1252.7333333333333,
			1240.6166666666666, 1417.85, 1229.8833333333332, 1262.2333333333333, 1247.6,
			1251.9, 1252.25, 1246.9833333333333, 1267.15, 1249.0166666666667, 1274.5833333333333,
			1241.6333333333332, 1276.1, 1244.8666666666666,
		},
		StopPoints: []float64{
			1234.5833333333333,
		},
	},
}

func TestMatchedSentinel(t *testing.T) {
	logger, _ := logging.GetLogger("SelfState")
	mockCtrl := gomock.NewController(t)
	database := mock_moira_alert.NewMockDatabase(mockCtrl)

	protector := Protector{}
	protector.Init(map[string]string{
		"k": "0.5",
	}, database, logger)
	values := protector.GetInitialValues()

	for _, sample := range samples {
		for _, point := range sample.Serie {
			database.EXPECT().GetNotifierState().Return("OK", nil)
			newValues := []int64{0, int64(point)}
			if degraded := protector.IsStateDegraded(values, newValues); degraded {
				fmt.Println(values)
				break
			}
			values = newValues
		}
	}
}
