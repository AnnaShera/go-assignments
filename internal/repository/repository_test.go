package repository

import "testing"

// TestPagination_Normalize drives Pagination.Normalize, which the
// List* repository methods apply before querying: Limit <= 0 defaults to
// defaultPageLimit (covers both the zero-value "no params given" case and
// an explicit non-positive value), a Limit above maxPageLimit is clamped
// down rather than rejected, and Offset is passed through unchanged since
// a negative offset is a client mistake the handler layer rejects
// outright rather than something the repository silently corrects.
func TestPagination_Normalize(t *testing.T) {
	tests := []struct {
		name       string
		in         Pagination
		wantLimit  int
		wantOffset int
	}{
		{name: "zero value defaults limit", in: Pagination{}, wantLimit: defaultPageLimit, wantOffset: 0},
		{name: "negative limit defaults", in: Pagination{Limit: -5}, wantLimit: defaultPageLimit, wantOffset: 0},
		{name: "limit within range preserved", in: Pagination{Limit: 10}, wantLimit: 10, wantOffset: 0},
		{name: "limit above cap is clamped", in: Pagination{Limit: 10_000}, wantLimit: maxPageLimit, wantOffset: 0},
		{name: "limit exactly at cap preserved", in: Pagination{Limit: maxPageLimit}, wantLimit: maxPageLimit, wantOffset: 0},
		{name: "offset preserved as given", in: Pagination{Limit: 10, Offset: 30}, wantLimit: 10, wantOffset: 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.Normalize()
			if got.Limit != tt.wantLimit {
				t.Errorf("expected limit %d, got %d", tt.wantLimit, got.Limit)
			}
			if got.Offset != tt.wantOffset {
				t.Errorf("expected offset %d, got %d", tt.wantOffset, got.Offset)
			}
		})
	}
}
