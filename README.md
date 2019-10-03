# Moira 2.0  [![Build Status](https://travis-ci.org/moira-alert/moira.svg?branch=master)](https://travis-ci.org/moira-alert/moira) [![Coverage Status](https://coveralls.io/repos/github/moira-alert/moira/badge.svg?branch=master)](https://coveralls.io/github/moira-alert/moira?branch=master) [![Documentation Status](https://readthedocs.org/projects/moira/badge/?version=latest)](http://moira.readthedocs.io/en/latest/?badge=latest) [![Telegram](https://img.shields.io/badge/telegram-join%20chat-3796cd.svg)](https://t.me/moira_alert) [![Go Report Card](https://goreportcard.com/badge/github.com/moira-alert/moira)](https://goreportcard.com/report/github.com/moira-alert/moira)

Moira is a real-time alerting tool, based on [Graphite](https://graphite.readthedocs.io) data.

## Installation

Docker Compose is the easiest way to try:

```bash
git clone https://github.com/moira-alert/docker-compose.git
cd docker-compose
docker-compose pull
docker-compose up
```

See more on our [documentation page](https://moira.readthedocs.io/en/latest/installation/index.html).

Feed data in Graphite format to `localhost:2003`:

```bash
echo "local.random.diceroll 4 `date +%s`" | nc localhost 2003
```

Configure triggers at `localhost:8080` using your browser.

Other installation methods are available, see [documentation](https://moira.readthedocs.io/en/latest/installation/index.html).

## Contribution

Check our [contribution guideline](.github/CONTRIBUTING.md)

## Getting Started

See our [user guide](https://moira.readthedocs.io/en/latest/user_guide/index.html) that is based on a number of real-life scenarios, from simple and universal to complicated and specific.

## What is in the other repositories

Code in this repository is the backend part of Moira monitoring application.

* [web2.0](https://github.com/moira-alert/web2.0) is the frontend part.
* [doc](https://github.com/moira-alert/doc) is the documentation (hosted on [Read the Docs](https://moira.readthedocs.io)).
* [moira-trigger-role](https://github.com/moira-alert/moira-trigger-role) is the Ansible role you can use to manage triggers.
* [python-moira-client](https://github.com/moira-alert/python-moira-client) is the Python API client.

## Contact us

If you have any questions, you can ask us on [Telegram](https://t.me/moira_alert).

## Thanks

![SKB Kontur](https://kontur.ru/theme/ver-1652188951/common/images/logo_english.png)

Moira was originally developed and is supported by [SKB Kontur](https://kontur.ru/eng/about), a B2G company based in Ekaterinburg, Russia. We express gratitude to our company for encouraging us to opensource Moira and for giving back to the community that created [Graphite](https://graphite.readthedocs.io) and many other useful DevOps tools.
