package db

import (
	"meguca/auth"
	"meguca/common"
	"meguca/config"
	"meguca/test"
	"testing"
	"time"
)

func TestCaptchas(t *testing.T) {
	assertTableClear(t, "failed_captchas", "last_solved_captchas", "boards",
		"accounts", "spam_scores")
	writeAllBoard(t)
	config.Set(config.Configs{
		CaptchaTags: config.Defaults.CaptchaTags,
		Public: config.Public{
			Captcha: true,
		},
	})
	err := auth.LoadCaptchaServices()
	if err != nil {
		t.Fatal(err)
	}
	const ip = "::1"

	type testCase struct {
		name      string
		captcha   auth.Captcha
		hasSolved bool
		err       error
	}

	c1, err := auth.CreateTestCaptcha()
	if err != nil {
		t.Fatal(err)
	}
	c2, err := auth.CreateTestCaptcha()
	if err != nil {
		t.Fatal(err)
	}
	cases := []testCase{
		{"invalid", auth.Captcha{}, false, common.ErrInvalidCaptcha},
		{"valid", c1, true, nil},
		{"upsert last solved table", c2, true, nil},
	}
	for i := 1; i < incorrectCaptchaLimit-1; i++ {
		cases = append(cases, testCase{"invalid", auth.Captcha{}, true,
			common.ErrInvalidCaptcha})
	}
	cases = append(cases, testCase{"bot detection", auth.Captcha{}, true,
		common.ErrBanned})

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			err = ValidateCaptcha(c.captcha, ip)
			test.AssertDeepEquals(t, err, c.err)

			for _, dur := range [...]time.Duration{time.Hour, time.Minute} {
				has, err := SolvedCaptchaRecently(ip, dur)
				if err != nil {
					t.Fatal(err)
				}
				test.AssertDeepEquals(t, has, c.hasSolved)
			}
		})
	}

	err = expireLastSolvedCaptchas()
	if err != nil {
		t.Fatal(err)
	}
}
