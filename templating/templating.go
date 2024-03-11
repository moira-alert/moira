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

func date(unixTime int64) string {
	return time.Unix(unixTime, 0).Format(eventTimeFormat)
}

func formatDate(unixTime int64, format string) string {
	return time.Unix(unixTime, 0).Format(format)
}

func filterKeys(source template.FuncMap, keys []string) template.FuncMap {
	result := template.FuncMap{}
	for _, key := range keys {
		if value, ok := source[key]; ok {
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

var sprigFuncMap = filterKeys(sprig.FuncMap(), []string{
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

	// Advanced functions
	"uuidv4",
})

type Populater interface {
	Populate(template string) (string, error)
}

func populate(tmpl string, data any) (populatedTemplate string, err error) {
	defer func() {
		if errRecover := recover(); errRecover != nil {
			populatedTemplate = tmpl
			err = fmt.Errorf("PANIC in populate: %v, tmpl: %s, provided data: %#v", errRecover, tmpl, data)
		}
	}()

	buffer := bytes.Buffer{}

	template := template.New("populate-template").Funcs(sprigFuncMap).Funcs(funcMap)

	if template, err = template.Parse(tmpl); err != nil {
		return tmpl, err
	}

	if err = template.Execute(&buffer, data); err != nil {
		return tmpl, err
	}

	return strings.TrimSpace(buffer.String()), nil
}
