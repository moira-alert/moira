package redis

import (
	"fmt"
	"sync"
	"time"

	"github.com/moira-alert/moira"

	"github.com/FZambia/sentinel"
	"github.com/gomodule/redigo/redis"
)

// PoolDialer hides details of how connections are created and tested in a pool
type PoolDialer interface {
	// Dial creates a connection
	Dial() (redis.Conn, error)
	// Test helps to check if a connection
	Test(c redis.Conn, t time.Time) error
}

// DirectPoolDialer connects directly to Redis
type DirectPoolDialer struct {
	serverAddress string
	db            int
	dialTimeout   time.Duration
}

// Dial connects directly to the server
func (dialer *DirectPoolDialer) Dial() (redis.Conn, error) {
	return redis.Dial(
		"tcp",
		dialer.serverAddress,
		redis.DialDatabase(dialer.db),
		redis.DialConnectTimeout(dialer.dialTimeout),
	)
}

// Test checks the connection by sending PING to the server
func (dialer *DirectPoolDialer) Test(c redis.Conn, t time.Time) error {
	_, err := c.Do("PING")
	return err
}

//SentinelPoolDialerConfig provides options to configure SentinelPoolDialer
type SentinelPoolDialerConfig struct {
	MasterName        string
	SentinelAddresses []string
	DB                int
	DialTimeout       time.Duration
}

//NewSentinelPoolDialer returns new SentinelPoolDialer
func NewSentinelPoolDialer(logger moira.Logger, config SentinelPoolDialerConfig) *SentinelPoolDialer {
	dialer := &SentinelPoolDialer{
		logger: logger,
		sentinel: &sentinel.Sentinel{
			Addrs:      config.SentinelAddresses,
			MasterName: config.MasterName,
			Dial: func(addr string) (redis.Conn, error) {
				return redis.Dial(
					"tcp",
					addr,
					redis.DialConnectTimeout(config.DialTimeout),
				)
			},
		},
		config: config,
	}
	go dialer.discoverLoop()
	return dialer
}

// SentinelPoolDialer connects directly to Redis through sentinels
type SentinelPoolDialer struct {
	logger          moira.Logger
	sentinel        *sentinel.Sentinel
	config          SentinelPoolDialerConfig
	lastMasterMutex sync.Mutex
	lastMaster      string
}

// Dial finds the master and connects to it
func (dialer *SentinelPoolDialer) Dial() (redis.Conn, error) {
	masterAddr, err := dialer.sentinel.MasterAddr()
	if err != nil {
		return nil, err
	}

	dialer.refreshLastMaster(masterAddr)

	return redis.Dial(
		"tcp",
		masterAddr,
		redis.DialDatabase(dialer.config.DB),
		redis.DialConnectTimeout(dialer.config.DialTimeout),
	)
}

// Test checks if connection is alive and connected to the master
func (dialer *SentinelPoolDialer) Test(c redis.Conn, t time.Time) error {
	if !sentinel.TestRole(c, "master") {
		return fmt.Errorf("failed master role check")
	}
	return nil
}

//NewSentinelPoolDialer returns new SentinelPoolDialer
func NewSentinelSlavePoolDialer(sentinelDialer *SentinelPoolDialer) *SentinelSlavePoolDialer {
	slaveDialer := &SentinelSlavePoolDialer{
		SentinelPoolDialer: sentinelDialer,
	}
	return slaveDialer
}

// SentinelSlavePoolDialer connects to Redis via sentinel prioritizing slave servers
type SentinelSlavePoolDialer struct {
	*SentinelPoolDialer
}

// Dial tries connecting to slaves
// If there are no slaves available, a connection to master is returned
func (dialer *SentinelSlavePoolDialer) Dial() (redis.Conn, error) {
	slaves, err := dialer.sentinel.SlaveAddrs()
	if err != nil {
		return nil, err
	}
	if len(slaves) == 0 {
		dialer.logger.Debug("No redis slaves available, connecting to master")
		return dialer.SentinelPoolDialer.Dial()
	}

	var conn redis.Conn
	for _, slaveAddr := range slaves {
		conn, err = redis.Dial(
			"tcp",
			slaveAddr,
			redis.DialDatabase(dialer.config.DB),
			redis.DialConnectTimeout(dialer.config.DialTimeout),
		)
		if err == nil {
			dialer.logger.Debugf("Connected to slave node %s", slaveAddr)
			break
		} else {
			dialer.logger.Warningf("Connecting to slave %s failed, error: %s", slaveAddr, err.Error())
		}
	}
	if err != nil {
		return dialer.SentinelPoolDialer.Dial()
	}

	// required for redis cluster, but will fail for simple replicas
	_, err = redis.String(conn.Do("READONLY"))
	if err != nil && err.Error() != "ERR This instance has cluster support disabled" {
		dialer.logger.Warning("Switching to readonly mode failed, error: %s", err.Error())
	}

	return conn, nil
}

// Test checks if connection is alive
func (dialer *SentinelSlavePoolDialer) Test(c redis.Conn, t time.Time) error {
	return c.Err()
}

func (dialer *SentinelPoolDialer) discoverLoop() {
	checkTicker := time.NewTicker(30 * time.Second)
	defer checkTicker.Stop()

	for range checkTicker.C {
		if err := dialer.sentinel.Discover(); err != nil {
			dialer.logger.Error(err)
		}
	}
}

func (dialer *SentinelPoolDialer) refreshLastMaster(master string) {
	dialer.lastMasterMutex.Lock()
	defer dialer.lastMasterMutex.Unlock()

	if master != dialer.lastMaster {
		dialer.logger.Infof("Redis master discovered: %s", master)
		dialer.lastMaster = master
	}
}
