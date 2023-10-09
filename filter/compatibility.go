package filter

type Compatibility struct {
	RegexTreatment  RegexTreatment
	SingleStarMatch SingleStarMatch
}

type RegexTreatment bool

const (
	StrictStartMatch RegexTreatment = false
	LooseStartMatch  RegexTreatment = true
)

type SingleStarMatch bool

const (
	SingleStarMatchDisabled    SingleStarMatch = false
	SingleStarMatchAllExisting SingleStarMatch = true
)
