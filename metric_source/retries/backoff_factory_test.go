package retries

import (
	"sync"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"

	. "github.com/smartystreets/goconvey/convey"
)

const (
	testInitialInterval     = time.Millisecond * 5
	testRandomizationFactor = 0.0
	testMultiplier          = 2.0
	testMaxInterval         = time.Millisecond * 40
)

func TestExponentialBackoffFactory(t *testing.T) {
	Convey("Test ExponentialBackoffFactory", t, func() {
		conf := Config{
			InitialInterval:     testInitialInterval,
			RandomizationFactor: testRandomizationFactor,
			Multiplier:          testMultiplier,
			MaxInterval:         testMaxInterval,
		}

		Convey("with MaxRetriesCount != 0 and MaxElapsedTime = 0", func() {
			Convey("with retry interval always lower then config.MaxInterval", func() {
				conf.MaxRetriesCount = 3
				defer func() {
					conf.MaxRetriesCount = 0
				}()

				expectedBackoffs := []time.Duration{
					testInitialInterval,
					testInitialInterval * testMultiplier,
					testInitialInterval * 4.0,
					backoff.Stop,
					backoff.Stop,
					backoff.Stop,
				}

				factory := NewExponentialBackoffFactory(conf)

				b := factory.NewBackOff()

				for i := range expectedBackoffs {
					So(b.NextBackOff(), ShouldEqual, expectedBackoffs[i])
				}
			})

			Convey("with retry interval becomes config.MaxInterval", func() {
				conf.MaxRetriesCount = 6
				defer func() {
					conf.MaxRetriesCount = 0
				}()

				expectedBackoffs := []time.Duration{
					testInitialInterval,
					testInitialInterval * testMultiplier,
					testInitialInterval * 4.0,
					testMaxInterval,
					testMaxInterval,
					testMaxInterval,
					backoff.Stop,
					backoff.Stop,
					backoff.Stop,
				}

				factory := NewExponentialBackoffFactory(conf)

				b := factory.NewBackOff()

				for i := range expectedBackoffs {
					So(b.NextBackOff(), ShouldEqual, expectedBackoffs[i])
				}
			})
		})

		Convey("with MaxRetriesCount = 0 and MaxElapsedTime != 0", func() {
			conf.MaxElapsedTime = time.Second
			defer func() {
				conf.MaxElapsedTime = 0
			}()

			once := sync.Once{}

			expectedBackoffs := []time.Duration{
				testInitialInterval,
				backoff.Stop,
				backoff.Stop,
				backoff.Stop,
			}

			factory := NewExponentialBackoffFactory(conf)

			b := factory.NewBackOff()

			for i := range expectedBackoffs {
				So(b.NextBackOff(), ShouldEqual, expectedBackoffs[i])
				once.Do(func() {
					time.Sleep(conf.MaxElapsedTime)
				})
			}
		})

		Convey("with MaxRetriesCount != 0 and MaxElapsedTime != 0", func() {
			Convey("MaxRetriesCount performed retries before MaxElapsedTime passed", func() {
				conf.MaxElapsedTime = time.Second
				conf.MaxRetriesCount = 6
				defer func() {
					conf.MaxElapsedTime = 0
					conf.MaxRetriesCount = 0
				}()

				expectedBackoffs := []time.Duration{
					testInitialInterval,
					testInitialInterval * testMultiplier,
					testInitialInterval * 4.0,
					testMaxInterval,
					testMaxInterval,
					testMaxInterval,
					backoff.Stop,
					backoff.Stop,
					backoff.Stop,
				}

				factory := NewExponentialBackoffFactory(conf)

				b := factory.NewBackOff()

				for i := range expectedBackoffs {
					So(b.NextBackOff(), ShouldEqual, expectedBackoffs[i])
				}
			})

			Convey("MaxElapsedTime passed before MaxRetriesCount performed", func() {
				conf.MaxElapsedTime = time.Second
				conf.MaxRetriesCount = 6
				defer func() {
					conf.MaxElapsedTime = 0
					conf.MaxRetriesCount = 0
				}()

				expectedBackoffs := []time.Duration{
					testInitialInterval,
					backoff.Stop,
					backoff.Stop,
					backoff.Stop,
				}

				once := sync.Once{}

				factory := NewExponentialBackoffFactory(conf)

				b := factory.NewBackOff()

				for i := range expectedBackoffs {
					So(b.NextBackOff(), ShouldEqual, expectedBackoffs[i])
					once.Do(func() {
						time.Sleep(conf.MaxElapsedTime)
					})
				}
			})
		})
	})
}
