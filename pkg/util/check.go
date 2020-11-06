package util

import (
	"fmt"
	"regexp"
	"strings"
)

type NameRegexp struct {
	Regexp       *regexp.Regexp
	ErrMsg       string
	ExpectResult bool
}

type CheckName interface {
	GetNameRegexps() []*NameRegexp
}

var NameRegs = []*NameRegexp{
	{
		Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-]+$`),
		ErrMsg:       "name is not legal",
		ExpectResult: true,
	},
	{
		Regexp:       regexp.MustCompile(`(?i)^cmcc$|^ctcc$|^cucc$|^any$|^default$|^none$`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
	{
		Regexp:       regexp.MustCompile(`^-.|.-$`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
}

var DomainNameRegs = []*NameRegexp{
	{
		Regexp:       regexp.MustCompile(`^[0-9a-zA-Z-*.@]+$`),
		ErrMsg:       "name is not legal",
		ExpectResult: true,
	},
	{
		Regexp:       regexp.MustCompile(`(^_|-)`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
}

var ZoneNameRegs = []*NameRegexp{
	{
		Regexp:       regexp.MustCompile(`^[0-9a-zA-Z.]+$`),
		ErrMsg:       "name is not legal",
		ExpectResult: true,
	}, {
		Regexp:       regexp.MustCompile(`^\.+$`),
		ErrMsg:       "name is not legal",
		ExpectResult: false,
	},
}

func CheckNameValid(name string) error {
	for _, reg := range NameRegs {
		if ret := reg.Regexp.MatchString(name); ret != reg.ExpectResult {
			return fmt.Errorf(reg.ErrMsg)
		}
	}
	return nil
}

func CheckDomainNameValid(name string) error {
	for _, reg := range DomainNameRegs {
		if ret := reg.Regexp.MatchString(name); ret != reg.ExpectResult {
			return fmt.Errorf(reg.ErrMsg)
		}
	}
	return nil
}

func CheckZoneNameValid(name string) error {
	for _, reg := range ZoneNameRegs {
		if ret := reg.Regexp.MatchString(name); ret != reg.ExpectResult {
			return fmt.Errorf(reg.ErrMsg)
		}
	}
	return nil
}

func IsBrokenPipeErr(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "broken pipe") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset by peer")
}
