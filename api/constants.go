package api

// SortOrder represents the sorting order for entities.
type SortOrder string

const (
	// NoSortOrder means that entities may be unsorted.
	NoSortOrder SortOrder = ""
	// AscSortOrder means that entities should be ordered ascending (example: from 1 to 9).
	AscSortOrder SortOrder = "asc"
	// DescSortOrder means that entities should be ordered descending (example: from 9 to 1).
	DescSortOrder SortOrder = "desc"
)

const TeamIDVaildationErrorMsg string = "team ID contains invalid characters (allowed: 0-9, a-z, A-Z, -, ~, _, .)"
