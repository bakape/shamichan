package parser

import (
	"testing"

	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
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
			test.AssertEquals(t, name, c.name)
			test.AssertEquals(t, trip, c.trip)
		})
	}

	t.Run("name too long", func(t *testing.T) {
		t.Parallel()
		_, _, err := ParseName(test.GenString(common.MaxLenName + 1))
		if err != common.ErrNameTooLong {
			test.UnexpectedError(t, err)
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
			name: "no subject",
			err:  errNoSubject,
		},
		{
			name: "subject too long",
			in:   test.GenString(common.MaxLenSubject + 1),
			err:  common.ErrSubjectTooLong,
		},
		{
			name: "valid",
			in:   " abc ",
			out:  "abc",
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			sub, err := ParseSubject(c.in)
			test.AssertEquals(t, err, c.err)
			if c.err == nil {
				test.AssertEquals(t, sub, c.out)
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
			name: "no password",
			err:  errNoPostPassword,
		},
		{
			name: "too long",
			in:   test.GenString(common.MaxLenPostPassword + 1),
			err:  common.ErrPostPasswordTooLong,
		},
		{
			name: "valid",
			in:   test.GenString(common.MaxLenPostPassword),
			err:  nil,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			test.AssertEquals(t, VerifyPostPassword(c.in), c.err)
		})
	}
}
