package db

import (
	"net"
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/test"
)

func TestSpamScores(t *testing.T) {
	// Skip to avoid massive booru fetches on DB population
	test.SkipInCI(t)

	assertTableClear(
		t,
		"spam_scores",
		"last_solved_captchas",
		"boards",
		"accounts",
		``)

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
	now := time.Now().Round(time.Second)

	var sessions [4]auth.Base64Token
	for i := 0; i < 4; i++ {
		sessions[i] = genToken(t)
	}
	ips := [...]net.IP{
		net.ParseIP("226.209.126.221"),
		net.ParseIP("131.215.1.14"),
		net.ParseIP("99.188.17.210"),
		net.ParseIP("71.189.25.162"),
	}

	for i := 0; i < 4; i++ {
		c, err := auth.CreateTestCaptcha()
		if err != nil {
			t.Fatal(err)
		}
		err = ValidateCaptcha(c, sessions[i], ips[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	for i, score := range [...]int64{
		now.Add(-20 * spamDetectionThreshold).Unix(),
		now.Add(-5 * time.Second).Unix(),
		now.Add(10 * spamDetectionThreshold).Unix(),
	} {
		_, err = db.Exec(
			`insert into spam_scores (token, score)
			values ($1, $2)`,
			sessions[i+1][:], score,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	spamMu.Lock()
	spamScoreBuffer = make(map[auth.Base64Token]sessionData)
	for i := 0; i < 4; i++ {
		score := time.Second * 10
		if i == 3 {
			score = spamDetectionThreshold
		}
		spamScoreBuffer[sessions[i]] = sessionData{
			score: score,
			ip:    ips[i],
		}
	}
	err = flushSpamScores()
	spamMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name           string
		ip             net.IP
		session        auth.Base64Token
		needCaptcha    bool
		needCaptchaErr error
	}{
		{
			name:    "fresh write",
			ip:      ips[0],
			session: sessions[0],
		},
		{
			name:    "overwrite stale value",
			ip:      ips[1],
			session: sessions[1],
		},
		{
			name:        "increment DB value",
			ip:          ips[2],
			session:     sessions[2],
			needCaptcha: true,
		},
		{
			name:           "spam",
			ip:             ips[3],
			session:        sessions[3],
			needCaptchaErr: common.ErrSpamDected,
		},
		{
			name:        "no captcha solved in 3h",
			ip:          net.ParseIP("143.195.24.54"),
			session:     genToken(t),
			needCaptcha: true,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			need, err := NeedCaptcha(c.session, c.ip)
			if err != c.needCaptchaErr {
				test.UnexpectedError(t, err)
			}
			test.AssertEquals(t, need, c.needCaptcha)
		})
	}

	t.Run("clear score", func(t *testing.T) {
		err := recordValidCaptcha(sessions[2])
		if err != nil {
			t.Fatal(err)
		}
		need, err := NeedCaptcha(sessions[2], ips[2])
		if err != nil {
			t.Fatal(err)
		}
		test.AssertEquals(t, need, false)
	})

	t.Run("sync", func(t *testing.T) {
		err := syncSpamScores()
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestCaptchas(t *testing.T) {
	// Skip to avoid massive booru fetches on DB population
	test.SkipInCI(t)

	assertTableClear(t,
		"failed_captchas",
		"last_solved_captchas",
		"boards",
		"accounts",
		"spam_scores",
	)
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
	ip := net.ParseIP("::1")
	session := genToken(t)

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
			err = ValidateCaptcha(c.captcha, session, ip)
			test.AssertEquals(t, err, c.err)

			for _, dur := range [...]time.Duration{time.Hour, time.Minute} {
				has, err := SolvedCaptchaRecently(session, dur)
				if err != nil {
					t.Fatal(err)
				}
				test.AssertEquals(t, has, c.hasSolved)
			}
		})
	}
}

// Generate random auth.Base64Token
func genToken(t *testing.T) auth.Base64Token {
	t.Helper()

	b, err := auth.NewBase64Token()
	if err != nil {
		t.Fatal(err)
	}
	return b
}
