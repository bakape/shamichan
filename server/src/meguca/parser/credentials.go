package parser

import (
	"errors"
	"meguca/common"
	"meguca/config"
	"strings"

	"github.com/aquilax/tripcode"
)

var (
	errNoPostPassword = errors.New("no post password")
	errNoSubject      = errors.New("no subject")
)

// ParseName parses the name field into a name and tripcode, if any
func ParseName(name string) (string, string, error) {
	switch {
	case name == "":
		return name, name, nil
	case len(name) > common.MaxLenName:
		return "", "", common.ErrNameTooLong
	}
	err := IsPrintableString(name, false)
	if err != nil {
		return "", "", err
	}
	name = strings.TrimSpace(name)

	// #password for tripcodes and ##password for secure tripcodes
	firstHash := strings.IndexByte(name, '#')
	if firstHash > -1 {
		password := name[firstHash+1:]
		name = name[:firstHash]
		if password[0] == '#' {
			trip := tripcode.SecureTripcode(password[1:], config.Get().Salt)
			return name, trip, nil
		}
		return name, tripcode.Tripcode(password), nil
	}

	return name, "", nil
}

// ParseSubject verifies and trims a thread subject string
func ParseSubject(s string) (string, error) {
	switch {
	case s == "":
		return s, errNoSubject
	case len(s) > common.MaxLenSubject:
		return s, common.ErrSubjectTooLong
	}
	if err := IsPrintableString(s, false); err != nil {
		return s, err
	}
	return strings.TrimSpace(s), nil
}

// VerifyPostPassword verifies a post password exists does not surpass the
// maximum allowed length
func VerifyPostPassword(s string) error {
	switch {
	case s == "":
		return errNoPostPassword
	case len(s) > common.MaxLenPostPassword:
		return common.ErrPostPasswordTooLong
	default:
		return IsPrintableString(s, false)
	}
}
