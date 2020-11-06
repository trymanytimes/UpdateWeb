package resource

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
)

func TestAssetValid(t *testing.T) {
	cases := []struct {
		ip        string
		telephone string
		isValid   bool
	}{
		{"2001:503:ba3e::1", "11111111111", true},
		{"1.1.1.1", "12111111111", true},
		{"1.1.1.1.1", "12111111111", false},
		{"1.1.1.1", "1a111111111", false},
		{"1.1.1.1", "1", false},
		{"1.1.1.1", "1111111111111111111111", false},
	}

	for _, c := range cases {
		p := &Asset{
			IP:        c.ip,
			Telephone: c.telephone,
		}
		err := p.Validate()
		if c.isValid {
			ut.Assert(t, err == nil, "")
		} else {
			ut.Assert(t, err != nil, "")
		}
	}
}
