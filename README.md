Moira is a real-time alerting tool, based on Graphite data.
---
[![Build Status](https://travis-ci.org/moira-alert/moira-alert.svg?branch=master)](https://travis-ci.org/moira-alert/moira-alert)
[![Coverage Status](https://coveralls.io/repos/github/moira-alert/moira-alert/badge.svg?branch=master)](https://coveralls.io/github/moira-alert/moira-alert?branch=master)

Fast Start
----------

```
git clone https://github.com/moira-alert/moira.git
cd moira-alert
compose up
```

Get Moira
---------
Use one of four ways

1. Download rpm and deb packages from [GitHub Release Page](https://github.com/moira-alert/moira/releases/latest)

2. via go get
```
$ go get github.com/moira-alert/moira/cmd/moira
$ moira -default-config > /etc/moira.yml
$ moira -config=/etc/moira.yml
```
3. via docker
```
$ docker pull kontur/moira
$ docker run kontur/moira -default-config > /home/user/moira.yml
$ docker run \
    -p 2003:2003 \
    -p 8081:8081 \
    -v /home/user/moira.yml:/moira.yml \
    -v /home/user/storage-schemas.conf.json:/storage-schemas.conf \
    kontur/moira
```

4. Manualy
```
$ git clone https://github.com/moira-alert/moira.git
$ cd moira-alert
$ make build
```

Configuration
-------------

```
$ moira -default-config
redis:
  host: localhost
  port: "6379"
  dbid: 0
graphite:
  enabled: "false"
  uri: localhost:2003
  prefix: DevOps.Moira
  interval: 60s0ms
checker:
  enabled: "true"
  nodata_check_interval: 60s0ms
  check_interval: 5s0ms
  metrics_ttl: 3600
  stop_checking_interval: 30
  log_file: stdout
  log_level: debug
api:
  enabled: "true"
  listen: :8081
  log_file: stdout
  log_level: debug
filter:
  enabled: "true"
  listen: :2003
  retention-config: storage-schemas.conf
  log_file: stdout
  log_level: debug
notifier:
  enabled: "true"
  sender_timeout: 10s0ms
  resending_timeout: "24:00"
  senders: []
  moira_selfstate:
    enabled: "false"
    redis_disconect_delay: 30
    last_metric_received_delay: 60
    last_check_delay: 60
    contacts: []
    notice_interval: 300
  log_file: stdout
  log_level: debug
  front_uri: https://moira.example.com
log_file: stdout
log_level: debug
```

License
-------

This code is licensed under the GPLv3 [license](LICENSE.md).
