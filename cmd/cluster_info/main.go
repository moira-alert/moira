package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/moira-alert/moira"
	"github.com/moira-alert/moira/database/redis"
	logging "github.com/moira-alert/moira/logging/zerolog_adapter"
)

func makeDb() (moira.Logger, *redis.DbConnector) {
	logger, err := logging.ConfigureLog("stdout", "debug", "test", true)
	if err != nil {
		panic("Failed to init logger " + err.Error())
	}

	database := redis.NewDatabase(logger, redis.DatabaseConfig{
		Addrs:        []string{"localhost:6370", "localhost:6371", "localhost:6372", "localhost:6373", "localhost:6374", "localhost:6375"},
		MetricsTTL:   time.Hour * 3,
		DialTimeout:  time.Second * 1,
		ReadTimeout:  time.Second * 1,
		WriteTimeout: time.Second * 1,
		// MaxRetries:    15,
		ReadOnly:      true,
		RouteRandomly: true,
	}, redis.NotificationHistoryConfig{}, redis.NotificationConfig{}, "test")

	return logger, database
}

var namesByPort = map[string]string{
	"6370": "redis_node_0",
	"6371": "redis_node_1",
	"6372": "redis_node_2",
	"6373": "redis_node_3",
	"6374": "redis_node_4",
	"6375": "redis_node_5",
}

type Node struct {
	port       string
	master     bool
	masterPort string
}

// redis_node_0 redis_node_3 redis_node_5

func main() {
	_, database := makeDb()

	cmd := database.Client().ClusterNodes(context.Background())
	resp := cmd.Val()
	lines := strings.Split(resp, "\n")

	nodes := map[string]Node{}
	masters := []string{}

	fmt.Printf("%v\n\n", resp)
	for _, line := range lines {
		if line == "" {
			continue
		}
		args := strings.Split(line, " ")
		id := args[0]
		host := strings.Split(args[1], "@")[0]
		port := strings.Split(host, ":")[1]
		master := args[2][0] != 's'

		if master {
			masters = append(masters, id)
		}

		nodes[id] = Node{
			port:   port,
			master: master,
		}
	}

	for _, line := range lines {
		if line == "" {
			continue
		}
		args := strings.Split(line, " ")
		id := args[0]

		node := nodes[id]
		if !node.master {
			masterId := args[3]

			node.masterPort = nodes[masterId].port

			fmt.Printf("S: %s -> %s\n", namesByPort[node.port], namesByPort[node.masterPort])
		} else {

			fmt.Printf("M: %s\n", namesByPort[node.port])
		}
	}
}
