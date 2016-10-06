package parser

import (
	"errors"
	"strings"

	"github.com/aquilax/tripcode"
	"github.com/bakape/meguca/config"
)

var (
	errNoPostPassword     = errors.New("no post password")
	errNoSubject          = errors.New("no subject")
	errNameTooLong        = ErrTooLong("name")
	errSubjectTooLong     = ErrTooLong("subject")
	errPostPaswordTooLong = ErrTooLong("post password")
)

// ParseName parses the name field into a name and tripcode, if any
func ParseName(name string) (string, string, error) {
	if name == "" {
		return name, name, nil
	}
	if len(name) > maxLengthName {
		return "", "", errNameTooLong
	}
	name = stripAndTrim(name)

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
	if s == "" {
		return s, errNoSubject
	}
	if len(s) > maxLengthSubject {
		return s, errSubjectTooLong
	}
	return stripAndTrim(s), nil
}

// FormatEmail validates and checks
func FormatEmail(email string) string {
	if email == "" || email == "sage" || len(email) > maxLengthEmail {
		return ""
	}
	return stripAndTrim(email)
}

// VerifyPostPassword verifies a post password exists does not surpass the
// maximum allowed length
func VerifyPostPassword(s string) error {
	if s == "" {
		return errNoPostPassword
	}
	if len(s) > maxLengthPostPassword {
		return errPostPaswordTooLong
	}
	return nil
}
