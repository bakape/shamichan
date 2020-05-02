package db

import (
	"context"
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

	clearTables(
		t,
		"spam_scores",
		"last_solved_captchas",
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
	now := time.Now().Round(time.Second)

	var users [5]uint64
	for i := 0; i < 5; i++ {
		users[i], _ = insertSamplePubKey(t)
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 4; i++ {
		c, err := auth.CreateTestCaptcha()
		if err != nil {
			t.Fatal(err)
		}
		err = ValidateCaptcha(context.Background(), c, users[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	threshold := now.Add(spamDetectionThreshold)
	for i, score := range [...]time.Time{
		threshold.Add(-20 * spamDetectionThreshold),
		threshold.Add(-1 * time.Second),
		threshold.Add(10 * spamDetectionThreshold),
	} {
		_, err = db.Exec(
			context.Background(),
			`insert into spam_scores (public_key, expires)
			values ($1, $2)`,
			users[i+1],
			score,
		)
		if err != nil {
			t.Fatal(err)
		}
	}

	spamMu.Lock()
	spamScoreBuffer = make(map[uint64]time.Duration)
	for i := 0; i < 4; i++ {
		score := time.Second * 10
		if i == 3 {
			score = spamDetectionThreshold
		}
		spamScoreBuffer[users[i]] = score
	}
	err = flushSpamScores()
	spamMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}

	cases := [...]struct {
		name           string
		pubKey         uint64
		needCaptcha    bool
		needCaptchaErr error
	}{
		{
			name:   "fresh write",
			pubKey: users[0],
		},
		{
			name:   "overwrite stale value",
			pubKey: users[1],
		},
		{
			name:        "increment DB value",
			pubKey:      users[2],
			needCaptcha: true,
		},
		{
			name:        "no captcha solved in 3h",
			pubKey:      users[4],
			needCaptcha: true,
		},
	}

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			need, err := NeedCaptcha(context.Background(), c.pubKey)
			if err != c.needCaptchaErr {
				test.UnexpectedError(t, err)
			}
			test.AssertEquals(t, need, c.needCaptcha)
		})
	}

	t.Run("clear score", func(t *testing.T) {
		err := recordValidCaptcha(context.Background(), users[2])
		if err != nil {
			t.Fatal(err)
		}
		need, err := NeedCaptcha(context.Background(), users[2])
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

	clearTables(t,
		"last_solved_captchas",
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
	user, _ := insertSamplePubKey(t)

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
		{
			name: "invalid",
			err:  common.ErrInvalidCaptcha,
		},
		{
			name:      "valid",
			captcha:   c1,
			hasSolved: true,
		},
		{
			name:      "upsert last solved table",
			captcha:   c2,
			hasSolved: true,
		},
	}
	for i := 1; i < 9; i++ {
		cases = append(cases, testCase{
			name:      "invalid",
			hasSolved: true,
			err:       common.ErrInvalidCaptcha,
		})
	}

	// TODO: Once bans are reimplemented
	// cases = append(cases, testCase{
	// 	name:      "bot detection",
	// 	hasSolved: true,
	// 	err:       common.ErrBanned,
	// })

	for i := range cases {
		c := cases[i]
		t.Run(c.name, func(t *testing.T) {
			err = ValidateCaptcha(context.Background(), c.captcha, user)
			test.AssertEquals(t, err, c.err)

			has, err := SolvedCaptchaRecently(context.Background(), user)
			if err != nil {
				t.Fatal(err)
			}
			test.AssertEquals(t, has, c.hasSolved)
		})
	}
}
