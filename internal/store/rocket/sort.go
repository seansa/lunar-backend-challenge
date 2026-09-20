package rocket

type SortField string

const (
	SortByChannel SortField = "id"
	SortByType    SortField = "type"
	SortByMission SortField = "mission"
	SortByStatus  SortField = "status"
)

type ListOptions struct {
	Sort       SortField
	Descending bool
}

func ParseSortField(value string) (SortField, bool) {
	switch SortField(value) {
	case SortByChannel, SortByType, SortByMission, SortByStatus:
		return SortField(value), true
	default:
		return "", false
	}
}
