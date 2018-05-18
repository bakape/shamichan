// Package mLog handles the log and it's handlers
package mLog

import (
	"sync"

	"meguca/config"

	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
	"github.com/go-playground/log/handlers/email"
)

const DefaultTimeFormat = "2006-01-02 15:04:05"

var (
	// Ensures no data races
	mutex sync.RWMutex

	// Console handler
	cLog *console.Console

	// Email handler
	eLog *email.Email
)

// Initialize the logger.
func Init(handler string) {
	mutex.Lock()

	switch handler {
		case "console":
			cLog = console.New(true)
			cLog.SetTimestampFormat(DefaultTimeFormat)
			log.AddHandler(cLog, log.AllLevels...)
			break
		case "email":
			conf := config.Get()
			eLog = email.New(conf.EmailErrSub, int(conf.EmailErrPort), conf.EmailErrMail, conf.EmailErrPass, conf.EmailErrMail, []string{conf.EmailErrMail})
			eLog.SetTimestampFormat(DefaultTimeFormat)
			log.AddHandler(eLog, log.ErrorLevel, log.PanicLevel, log.AlertLevel, log.FatalLevel)
			break
		default:
			log.Fatal("Invalid handler: ", handler)
	}

	mutex.Unlock()
}

// Update the logger.
func Update() {
	// TODO: https://github.com/go-playground/log/issues/19
}
