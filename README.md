# Moira 2.0 BETA [![Build Status](https://travis-ci.org/moira-alert/moira.svg?branch=master)](https://travis-ci.org/moira-alert/moira) [![Coverage Status](https://coveralls.io/repos/github/moira-alert/moira/badge.svg?branch=master)](https://coveralls.io/github/moira-alert/moira?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/moira-alert/moira)](https://goreportcard.com/report/github.com/moira-alert/moira)

Moira 2.0 is a completely rewritten version of:

1. Checker service in Go (instead of Python), based on [carbonapi](https://github.com/go-graphite/carbonapi) implementation.
2. Web service in React, with slightly different UI.

There is still some work to do before we can call it a release:

1. Update documentation with new screenshots and installation instructions. [Old documentation](https://moira.readthedocs.io) is still mostly relevant.
2. Add convenient ways to install Moira, like `docker-compose`.


## Temporary installation instructions (until we come up with something better)

Moira consists of 4 separate microservices (api, filter, checker and notifier) and a CLI application.
You'll need to install and launch all of them.

Choose one:

1. Download rpm and deb packages from [GitHub Release Page](https://github.com/moira-alert/moira/releases/latest).
Install as usual, and you'll get `moira-api`, `moira-checker`, `moira-filter` and `moira-notifier` services in your systemd.
Edit configs in `/etc/moira/` and start services.

2. Use Docker. You can mount volumes and override entrypoints to use custom configs. See `Dockerfile.*` for details.
```
$ docker pull moira/api
$ docker pull moira/filter
$ docker pull moira/checker
$ docker pull moira/notifier
```

3. Use Makefile and just get your binaries in `build` directory.
```
$ git clone https://github.com/moira-alert/moira.git
$ cd moira
$ make build
```
