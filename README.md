# Moira 2.0 [![Documentation Status](https://readthedocs.org/projects/moira/badge/?version=latest)](http://moira.readthedocs.io/en/latest/?badge=latest) [![Telegram](https://img.shields.io/badge/telegram-join%20chat-3796cd.svg)](https://t.me/moira_alert) [![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/moira-alert/moira?utm_source=badge&utm_medium=badge&utm_campaign=badge) [![Build Status](https://travis-ci.org/moira-alert/moira.svg?branch=master)](https://travis-ci.org/moira-alert/moira) [![Coverage Status](https://coveralls.io/repos/github/moira-alert/moira/badge.svg?branch=master)](https://coveralls.io/github/moira-alert/moira?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/moira-alert/moira)](https://goreportcard.com/report/github.com/moira-alert/moira)

Moira 2.0 is a completely rewritten version of:

1. Checker and API services in Go (instead of Python), based on [carbonapi](https://github.com/go-graphite/carbonapi) implementation.
2. Web service in React, with slightly different UI.

Code in this repository is a backend part of Moira monitoring application. Frontend part is [here][web2].

Documentation for the entire Moira project is available on [Read the Docs][readthedocs] site.

If you have any questions, you can ask us on [Gitter][gitter].

## Thanks

![SKB Kontur](https://kontur.ru/theme/ver-1652188951/common/images/logo_english.png)

Moira was originally developed and is supported by [SKB Kontur][kontur], a B2G company based in Ekaterinburg, Russia. We express gratitude to our company for encouraging us to opensource Moira and for giving back to the community that created [Graphite][graphite] and many other useful DevOps tools.


[web2]: https://github.com/moira-alert/web2.0
[readthedocs]: http://moira.readthedocs.io
[gitter]: https://gitter.im/moira-alert/moira
[kontur]: https://kontur.ru/eng/about
[graphite]: http://graphite.readthedocs.org
