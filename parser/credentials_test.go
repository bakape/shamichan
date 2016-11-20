package parser

import (
	"testing"

	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
	"github.com/bakape/meguca/common"
)

func TestParseName(t *testing.T) {
	(*config.Get()).Salt = "123"

	cases := [...]struct {
		testName, in, name, trip string
	}{
		{"empty", "", "", ""},
		{"name only", "name", "name", ""},
		{"trip only", "#test", "", ".CzKQna1OU"},
		{"name and trip", "name#test", "name", ".CzKQna1OU"},
		{"secure trip", "##test", "", "mb8h72.d9g"},
		{"name secure trip", "name##test", "name", "mb8h72.d9g"},
		{"with padding spaces", "  name##test ", "name", "mb8h72.d9g"},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.testName, func(t *testing.T) {
			t.Parallel()

			name, trip, err := ParseName(c.in)
			if err != nil {
				t.Fatal(err)
			}
			if name != c.name {
				LogUnexpected(t, c.name, name)
			}
			if trip != c.trip {
				LogUnexpected(t, c.trip, trip)
			}
		})
	}

	t.Run("name too long", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseName(genString(common.MaxLenName + 1))
		if err != common.ErrNameTooLong {
			UnexpectedError(t, err)
		}
	})
}

func TestParseSubject(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in, out string
		err           error
	}{
		{
			"no subject",
			"", "", errNoSubject,
		},
		{
			"subject too long",
			genString(common.MaxLenSubject + 1), "", common.ErrSubjectTooLong,
		},
		{
			"valid",
			" abc ", "abc", nil,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			sub, err := ParseSubject(c.in)
			if err != c.err {
				UnexpectedError(t, err)
			}
			if c.err == nil {
				if sub != c.out {
					LogUnexpected(t, c.out, sub)
				}
			}
		})
	}
}

func TestVerifyPostPassword(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in string
		err      error
	}{
		{
			"no password",
			"",
			errNoPostPassword,
		},
		{
			"too long",
			genString(common.MaxLenPostPassword + 1),
			common.ErrPostPasswordTooLong,
		},
		{
			"valid",
			genString(common.MaxLenPostPassword),
			nil,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			if err := VerifyPostPassword(c.in); err != c.err {
				UnexpectedError(t, err)
			}
		})
	}
}

func TestFormatEmail(t *testing.T) {
	t.Parallel()

	cases := [...]struct {
		name, in, out string
	}{
		{"empty", "", ""},
		{"normal", "foo", "foo"},
		{"too long", genString(common.MaxLenEmail + 1), ""},
	}
	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			if s := FormatEmail(c.in); s != c.out {
				LogUnexpected(t, c.out, s)
			}
		})
	}
}
