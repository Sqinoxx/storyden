package bug_report

//go:generate go run github.com/Southclaws/enumerator

type categoryEnum string

const (
	categoryBug        categoryEnum = "bug"
	categorySuggestion categoryEnum = "suggestion"
	categoryOther      categoryEnum = "other"
)

type statusEnum string

const (
	statusOpen     statusEnum = "open"
	statusPlanned  statusEnum = "planned"
	statusResolved statusEnum = "resolved"
	statusDeclined statusEnum = "declined"
)
