// Package mLog handles the log and it's handlers
package mLog

import (
	"sync"

	"meguca/config"

	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
	"github.com/go-playground/log/handlers/email"
)

type handler uint8

const (
	DefaultTimeFormat = "2006-01-02 15:04:05"

	Console handler = iota
	Email
)

var (
	// Ensures no data races
	rw sync.RWMutex

	// Ensure email handler is only added once
	once sync.Once

	// Console handler
	ConsoleHandler *console.Console

	// Email handler
	eLog *email.Email
)

// Initialize the logger.
func Init(h handler) {
	rw.Lock()
	defer rw.Unlock()

	switch h {
	case Console:
		ConsoleHandler = console.New(true)
		ConsoleHandler.SetTimestampFormat(DefaultTimeFormat)
		log.AddHandler(ConsoleHandler, log.AllLevels...)
	case Email:
		conf := config.Get()

		eLog = email.New(conf.EmailErrSub, int(conf.EmailErrPort),
			conf.EmailErrMail, conf.EmailErrPass, conf.EmailErrMail,
			[]string{conf.EmailErrMail})

		eLog.SetEnabled(conf.EmailErr)
		eLog.SetTimestampFormat(DefaultTimeFormat)

		if conf.EmailErr {
			once.Do(func() {
				log.AddHandler(eLog, log.ErrorLevel, log.PanicLevel,
					log.AlertLevel, log.FatalLevel)
			})
		}
	default:
		log.Fatal("Invalid mLog handler: ", h)
	}
}

// Update the logger.
func Update() {
	rw.Lock()
	defer rw.Unlock()

	conf := config.Get()

	eLog.SetEmailConfig(conf.EmailErrSub, int(conf.EmailErrPort),
		conf.EmailErrMail, conf.EmailErrPass, conf.EmailErrMail,
		[]string{conf.EmailErrMail})

	eLog.SetEnabled(conf.EmailErr)

	if conf.EmailErr {
		once.Do(func() {
			log.AddHandler(eLog, log.ErrorLevel, log.PanicLevel, log.AlertLevel,
				log.FatalLevel)
		})
	}
}
