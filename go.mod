module github.com/moira-alert/moira

go 1.16

require (
	github.com/FZambia/sentinel v1.1.0
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible
	github.com/PagerDuty/go-pagerduty v1.3.0
	github.com/aws/aws-sdk-go v1.35.4
	github.com/beevee/go-chart v2.0.2-0.20190523110126-273a59e48bcc+incompatible
	github.com/blend/go-sdk v2.0.0+incompatible // indirect
	github.com/blevesearch/bleve/v2 v2.1.0
	github.com/bwmarrin/discordgo v0.22.0
	github.com/carlosdp/twiliogo v0.0.0-20161027183705-b26045ebb9d1
	github.com/cespare/xxhash/v2 v2.1.1
	github.com/cyberdelia/go-metrics-graphite v0.0.0-20161219230853-39f87cc3b432
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/dustin/go-humanize v1.0.0
	github.com/fatih/color v1.9.0 // indirect
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-chi/render v1.0.1
	github.com/go-graphite/carbonapi v0.0.0-20201019162650-b789c0eaed8a
	github.com/go-graphite/protocol v0.4.3
	github.com/go-redis/redis/v8 v8.11.3
	github.com/go-redsync/redsync v1.4.2
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/golang/mock v1.6.0
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/go-querystring v1.0.1-0.20190318165438-c8c88dbee036 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/gotokatsuya/ipare v0.0.0-20161202043954-fd52c5b6c44b
	github.com/gregdel/pushover v0.0.0-20200820121613-505cfd60a340
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2-0.20190406162018-d3fcbee8e181 // indirect
	github.com/hashicorp/go-hclog v0.14.1 // indirect
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.7 // indirect
	github.com/karriereat/blackfriday-slack v0.1.0
	github.com/kr/text v0.2.0 // indirect
	github.com/lib/pq v1.8.0 // indirect
	github.com/lomik/zapwriter v0.0.0-20201002100138-f85a75186af0 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mitchellh/mapstructure v1.3.3 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/opsgenie/opsgenie-go-sdk-v2 v1.2.1
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.14.0 // indirect
	github.com/prometheus/procfs v0.2.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/rs/cors v1.7.0
	github.com/rs/zerolog v1.20.0
	github.com/russross/blackfriday/v2 v2.0.1
	github.com/sirupsen/logrus v1.7.0 // indirect
	github.com/slack-go/slack v0.8.1
	github.com/smartystreets/assertions v1.2.0
	github.com/smartystreets/goconvey v1.6.4
	github.com/spf13/afero v1.4.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1 // indirect
	github.com/stretchr/objx v0.3.0 // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/writeas/go-strip-markdown v2.0.1+incompatible
	github.com/xiam/to v0.0.0-20200126224905-d60d31e03561
	go.uber.org/automaxprocs v1.4.0
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/exp v0.0.0-20200924195034-c827fd4f18b9 // indirect
	golang.org/x/image v0.0.0-20200927104501-e162460cd6b5 // indirect
	gonum.org/v1/netlib v0.0.0-20200824093956-f0ca4b3a5ef5 // indirect
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b // indirect
	gopkg.in/gomail.v2 v2.0.0-20160411212932-81ebce5c23df
	gopkg.in/h2non/gock.v1 v1.0.15
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637
	gopkg.in/tucnak/telebot.v2 v2.3.4
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
)
