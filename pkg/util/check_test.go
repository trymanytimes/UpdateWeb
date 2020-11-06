package util

import (
	"testing"
)

func TestIsContainsSubnetOrIP(t *testing.T) {
	tests := []struct {
		ipEntireNet, ipSubnet string
		expected              bool
	}{
		{ipEntireNet: "2008::/32", ipSubnet: "2008::/32", expected: true},
		{ipEntireNet: "2008::/32", ipSubnet: "2007::/32", expected: false},
		{ipEntireNet: "2008::/32", ipSubnet: "2008:0:0:10::/60", expected: true},
		{ipEntireNet: "2008::/32", ipSubnet: "2008:0:0:12::/64", expected: true},
		{ipEntireNet: "2008::/32", ipSubnet: "2007:0:0:13::/64", expected: false},
	}

	for _, tt := range tests {
		result, err := PrefixContainsSubnetOrIP(tt.ipEntireNet, tt.ipSubnet)
		if err != nil {
			t.Errorf("get err:%s", err.Error())
		}
		if result != tt.expected {
			t.Errorf("PrefixContainsSubnetOrIP failed:expected:%t but get %t", tt.expected, result)
		}
	}
}

func TestCheckNameValid(t *testing.T) {
	names := []string{
		"a", "-", "123", "abc12", "any",
		"1.2.3", "1_b", "a-c", "_", "111", "]",
		"a-", "-a", "_a", "b_", "b-", "-b", "asd-", "-123-"}

	for _, name := range names {
		if err := CheckNameValid(name); err != nil {
			t.Errorf("invalid name:%s err:%s ", name, err.Error())
		}
	}
}

func TestDomainName(t *testing.T) {
	names := []string{"__", "_a", "a-b", "-_", "abc", "b_",
		"b-", "-n", "a*b", "@", "*", "-asd-", "-123-"}
	for _, name := range names {
		if err := CheckDomainNameValid(name); err != nil {
			t.Errorf("invalid name:%s err:%s ", name, err.Error())
		}
	}
}

func TestCheckZoneNameValid(t *testing.T) {
	names := []string{"*", ".", "@", "asd*.com", "*.com", "@.com", ".com", "com.*", "com.", "..", "com", "99"}
	for _, name := range names {
		if err := CheckZoneNameValid(name); err != nil {
			t.Errorf("invalid name:%s err:%s ", name, err.Error())
		}
	}
}
