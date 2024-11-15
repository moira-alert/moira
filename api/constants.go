package api

type SortOrder string

const (
	NoSortOrder   SortOrder = ""
	AscSortOrder  SortOrder = "asc"
	DescSortOrder SortOrder = "desc"
)
