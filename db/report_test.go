package db

import (
	"testing"

	"github.com/bakape/meguca/auth"
	. "github.com/bakape/meguca/test"
)

func TestReports(t *testing.T) {
	assertTableClear(t, "boards", "reports")
	writeSampleBoard(t)
	writeSampleThread(t)

	std := auth.Report{
		Target: 1,
		Board:  "a",
		Reason: "foo",
	}
	err := Report(std.Target, std.Board, std.Reason, "::1", false)
	if err != nil {
		t.Fatal(err)
	}

	res, err := GetReports(std.Board)
	if err != nil {
		t.Fatal(err)
	}
	// Sync dynamic fields
	std.ID = res[0].ID
	std.Created = res[0].Created
	AssertEquals(t, []auth.Report{std}, res)
}
