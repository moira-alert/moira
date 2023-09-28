package filter

type Compatibility struct {
	// Controls how regexps are treated.
	// StrictStartMatch treats /<regex>/ as /^<regex>.*$/
	// LooseStartMatch treats /<regex>/ as /^.*<regex>.*$/
	RegexTreatment RegexTreatment
}

type RegexTreatment bool

const (
	StrictStartMatch RegexTreatment = false
	LooseStartMatch  RegexTreatment = true
)
