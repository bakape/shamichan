package db

import (
	"testing"
	"time"

	"github.com/bakape/meguca/auth"
	"github.com/bakape/meguca/common"
	"github.com/bakape/meguca/config"
	. "github.com/bakape/meguca/test"
)

func TestSpamScores(t *testing.T) {
	config.Set(config.Configs{
		CaptchaTags: config.Defaults.CaptchaTags,
		Public: config.Public{
			Captcha: true,
		},
	})
	assertTableClear(t, "spam_scores", "last_solved_captchas", "boards",
		"accounts")
	writeAllBoard(t)
	err := auth.LoadCaptchaServices()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Round(time.Second)

	for _, ip := range [...]string{
		"226.209.126.221",
		"131.215.1.14",
		"99.188.17.210",
		"71.189.25.162",
	} {
		c, err := auth.CreateTestCaptcha()
		if err != nil {
			t.Fatal(err)
		}
		err = ValidateCaptcha(c, ip)
		if err != nil {
			t.Fatal(err)
		}
	}

	for ip, score := range map[string]int64{
		"131.215.1.14":  now.Add(-20 * spamDetectionThreshold).Unix(),
		"99.188.17.210": now.Add(-5 * time.Second).Unix(),
		"71.189.25.162": now.Add(10 * spamDetectionThreshold).Unix(),
	} {
		_, err = sq.Insert("spam_scores").
			Columns("ip", "score").
			Values(ip, score).
			Exec()
		if err != nil {
			t.Fatal(err)
		}
	}

	spamMu.Lock()
	spamScoreBuffer = map[string]time.Duration{
		"226.209.126.221": time.Second * 10,
		"131.215.1.14":    time.Second * 10,
		"99.188.17.210":   time.Second * 10,
		"71.189.25.162":   spamDetectionThreshold,
	}
	err = flushSpamScores()
	spamMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name, ip       string
		needCaptcha    bool
		needCaptchaErr error
	}{
		{"fresh write", "226.209.126.221", false, nil},
		{"overwrite stale value", "131.215.1.14", false, nil},
		{"increment DB value", "99.188.17.210", true, nil},
		{"spam", "71.189.25.162", false, common.ErrSpamDected},
		{"no captcha solved in 3h", "143.195.24.54", true, nil},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			need, err := NeedCaptcha(c.ip)
			if err != c.needCaptchaErr {
				UnexpectedError(t, err)
			}
			AssertDeepEquals(t, need, c.needCaptcha)
		})
	}

	t.Run("clear score", func(t *testing.T) {
		const ip = "99.188.17.210"
		err := resetSpamScore(ip)
		if err != nil {
			t.Fatal(err)
		}
		need, err := NeedCaptcha(ip)
		if err != nil {
			t.Fatal(err)
		}
		AssertDeepEquals(t, need, false)
	})

	t.Run("expiry", func(t *testing.T) {
		err := expireSpamScores()
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("sync", func(t *testing.T) {
		err := syncSpamScores()
		if err != nil {
			t.Fatal(err)
		}
	})
}
