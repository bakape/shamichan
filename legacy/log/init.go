// Package mlog handles the log and it's handlers
package mlog

import (
	"sync"

	"github.com/bakape/meguca/config"
	"github.com/go-playground/log"
	"github.com/go-playground/log/handlers/console"
	"github.com/go-playground/log/handlers/email"
	"gopkg.in/gomail.v2"
)

type handler uint8

const (
	// DefaultTimeFormat is the default time format
	DefaultTimeFormat = "2006-01-02 15:04:05"

	// Console handler is the console handler
	Console handler = iota
	// Email is the email handler
	Email
)

var (
	// Ensures no data races
	rw sync.RWMutex

	// Ensure email handler is only added once
	once sync.Once

	// ConsoleHandler is the console handler
	ConsoleHandler *console.Console

	// Email handler
	eLog *email.Email
)

// Init initializes the logger.
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
		setEmailHandler()
	default:
		log.Fatal("Invalid mlog handler: ", h)
	}
}

func setEmailHandler() {
	conf := config.Get()
	eLog.SetEmailConfig(conf.EmailErrSub, int(conf.EmailErrPort),
		conf.EmailErrMail, conf.EmailErrPass, conf.EmailErrMail,
		[]string{conf.EmailErrMail})
	eLog.SetEnabled(conf.EmailErr)
	eLog.SetFormatFunc(format)

	if conf.EmailErr {
		once.Do(func() {
			log.AddHandler(eLog, log.ErrorLevel, log.PanicLevel, log.AlertLevel,
				log.FatalLevel)
		})
	}
}

// Update the logger.
func Update() {
	rw.Lock()
	defer rw.Unlock()

	setEmailHandler()
}

func format(e *email.Email) email.Formatter {
	return func(entry log.Entry) *gomail.Message {
		addr := config.Get().EmailErrMail
		msg := gomail.NewMessage()
		msg.SetHeader("From", addr)
		msg.SetHeader("To", addr)
		msg.SetHeader("Subject", "meguca error")
		msg.SetBody("text/plain", entry.Message)
		return msg
	}
}
