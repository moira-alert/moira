package templating

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
)

const eventTimeFormat = "2006-01-02 15:04:05"

type notification struct {
	Trigger trigger
	Events  []Event
}

type Event struct {
	Metric         string
	MetricElements []string
	Timestamp      int64
	Value          *float64
	State          string
}

func date(unixTime int64) string {
	return time.Unix(unixTime, 0).Format(eventTimeFormat)
}

func formatDate(unixTime int64, format string) string {
	return time.Unix(unixTime, 0).Format(format)
}

func (event Event) TimestampDecrease(second int64) int64 {
	return event.Timestamp - second
}

func (event Event) TimestampIncrease(second int64) int64 {
	return event.Timestamp + second
}

type trigger struct {
	Name string `json:"name"`
}

func FilterKeys(source template.FuncMap, keys []string) template.FuncMap {
	result := template.FuncMap{}
	for _, key := range keys {
		value, found := source[key]
		if found {
			result[key] = value
		}
	}
	return result
}

var funcMap = template.FuncMap{
	"date":              date,
	"formatDate":        formatDate,
	"stringsReplace":    strings.Replace,
	"stringsToLower":    strings.ToLower,
	"stringsToUpper":    strings.ToUpper,
	"stringsTrimPrefix": strings.TrimPrefix,
	"stringsTrimSuffix": strings.TrimSuffix,
	"stringsSplit":      strings.Split,
}

var sprigFuncMap = FilterKeys(sprig.FuncMap(), []string {
	// Date functions
	"ago",
	"date",
	"date_in_zone",
	"date_modify",
	"dateInZone",
	"dateModify",
	"duration",
	"durationRound",
	"htmlDate",
	"htmlDateInZone",
	"must_date_modify",
	"mustDateModify",
	"mustToDate",
	"now",
	"toDate",
	"unixEpoch",

	// Strings
	"abbrev",
	"abbrevboth",
	"trunc",
	"trim",
	"upper",
	"lower",
	"title",
	"untitle",
	"substr",
	"repeat",
	"trimall",
	"trimAll",
	"trimSuffix",
	"trimPrefix",
	"nospace",
	"initials",
	"randAlphaNum",
	"randAlpha",
	"randAscii",
	"randNumeric",
	"swapcase",
	"shuffle",
	"snakecase",
	"camelcase",
	"kebabcase",
	"wrap",
	"wrapWith",
	"contains",
	"hasPrefix",
	"hasSuffix",
	"quote",
	"squote",
	"cat",
	"indent",
	"nindent",
	"replace",
	"plural",
	"sha1sum",
	"sha256sum",
	"adler32sum",
	"toString",

	// Wrap Atoi to stop errors.
	"atoi",
	"int64",
	"int",
	"float64",
	"toDecimal",
	"toStrings",
	
	// String Slice Functions
	"split",
	"splitList",
	"splitn",
	"join",
	"sortAlpha",
	
	// Integer Slice Functions
	"seq",
	"until",
	"untilStep",

	// Integer Math Functions
	"add1",
	"add",
	"sub",
	"div",
	"mod",
	"mul",
	"randInt",
	"addf",
	"subf",
	"divf",
	"mulf",
	"biggest",
	"max",
	"min",
	"maxf",
	"minf",
	"ceil",
	"floor",
	"round",

	// Defaults
	"default",
	"empty",
	"coalesce",
	"all",
	"any",
	"compact",
	"mustCompact",
	"fromJson",
	"toJson",
	"toPrettyJson",
	"toRawJson",
	"mustFromJson",
	"mustToJson",
	"mustToPrettyJson",
	"mustToRawJson",
	"ternary",
	"deepCopy",
	"mustDeepCopy",

	// Data Structures
	"list",
	"dict",
	"get",
	"set",
	"unset",
	"hasKey",
	"pluck",
	"keys",
	"pick",
	"omit",
	"merge",
	"mergeOverwrite",
	"mustMerge",
	"mustMergeOverwrite",
	"values",

	// List
	"append",
	"mustAppend",
	"push",
	"mustPush",
	"prepend",
	"mustPrepend",
	"first",
	"mustFirst",
	"rest",
	"mustRest",
	"last",
	"mustLast",
	"initial",
	"mustInitial",
	"reverse",
	"mustReverse",
	"uniq",
	"mustUniq",
	"without",
	"mustWithout",
	"has",
	"mustHas",
	"slice",
	"mustSlice",
	"concat",
	"dig",
	"chunk",
	"mustChunk",

	// Regex
	"regexMatch",
	"mustRegexMatch",
	"regexFindAll",
	"mustRegexFindAll",
	"regexFind",
	"mustRegexFind",
	"regexReplaceAll",
	"mustRegexReplaceAll",
	"regexReplaceAllLiteral",
	"mustRegexReplaceAllLiteral",
	"regexSplit",
	"mustRegexSplit",
	"regexQuoteMeta",
})

func Populate(name, description string, events []Event) (desc string, err error) {
	defer func() {
		if errRecover := recover(); errRecover != nil {
			desc = description
			err = fmt.Errorf("PANIC in populate: %v, Trigger name: %s, desc: %s, events:%#v",
				err, name, description, events)
		}
	}()

	buffer := bytes.Buffer{}

	dataToExecute := notification{
		Trigger: trigger{Name: name},
		Events:  events,
	}

	triggerTemplate := template.New("populate-description").Funcs(sprigFuncMap).Funcs(funcMap)
	triggerTemplate, err = triggerTemplate.Parse(description)
	if err != nil {
		return description, err
	}

	err = triggerTemplate.Execute(&buffer, dataToExecute)
	if err != nil {
		return description, err
	}

	return strings.TrimSpace(buffer.String()), nil
}
