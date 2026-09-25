package pagination

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		limit      int
		wantPage   int
		wantLimit  int
		wantOffset int
	}{
		{name: "Defaults when empty", page: 0, limit: 0, wantPage: 1, wantLimit: DefaultLimit, wantOffset: 0},
		{name: "Third page of 10", page: 3, limit: 10, wantPage: 3, wantLimit: 10, wantOffset: 20},
		{name: "Limit above max is clamped", page: 1, limit: 500, wantPage: 1, wantLimit: MaxLimit, wantOffset: 0},
		{name: "Negative page becomes first", page: -4, limit: 10, wantPage: 1, wantLimit: 10, wantOffset: 0},
	}

	for _, tableTest := range tests {
		t.Run(tableTest.name, func(t *testing.T) {
			page, limit, offset := Normalize(tableTest.page, tableTest.limit)

			if page != tableTest.wantPage || limit != tableTest.wantLimit || offset != tableTest.wantOffset {
				t.Fatalf("got (%d, %d, %d), want (%d, %d, %d)",
					page, limit, offset, tableTest.wantPage, tableTest.wantLimit, tableTest.wantOffset)
			}
		})
	}
}
