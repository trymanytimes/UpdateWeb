package handler

import "testing"

func TestRecombineSlices(tt *testing.T) {
	tests := []struct {
		s1       []string
		s2       []string
		isDelete bool
		expect   []string
	}{
		{s1: []string{"a1", "a2"}, s2: []string{"a1"}, isDelete: true, expect: []string{"a2"}},
		{s1: []string{"a1", "a2"}, s2: []string{"a1", "a3"}, isDelete: false, expect: []string{"a1", "a2", "a3"}},
		{s1: []string{"a1", "a2", "a1"}, s2: []string{}, isDelete: false, expect: []string{"a1", "a2"}},
		{s1: []string{"a1"}, s2: []string{"a1"}, isDelete: true, expect: []string{}},
	}

	for _, t := range tests {
		result := recombineSlices(t.s1, t.s2, t.isDelete)
		tt.Logf("expected:%+v result %+v \n", t.expect, result)
	}
}
