package rocket

import "testing"

func TestParseSortField(t *testing.T) {
	tests := []struct {
		value     string
		want      SortField
		wantValid bool
	}{
		{value: "id", want: SortByChannel, wantValid: true},
		{value: "type", want: SortByType, wantValid: true},
		{value: "mission", want: SortByMission, wantValid: true},
		{value: "status", want: SortByStatus, wantValid: true},
		{value: "", wantValid: false},
		{value: "speed", wantValid: false},
		{value: "ID", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			field, valid := ParseSortField(tt.value)

			if valid != tt.wantValid {
				t.Fatalf("expected valid=%t, got %t", tt.wantValid, valid)
			}
			if valid && field != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, field)
			}
		})
	}
}
