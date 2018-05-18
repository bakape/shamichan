package geoip

import (
	"net"
	"strings"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"github.com/go-playground/log"
)

// nil, if database not loaded
var db *maxminddb.Reader

var NY *time.Location

func init() {
	NY, _ = time.LoadLocation("America/New_York")
}

func Load() error {
	var err error
	db, err = maxminddb.Open("GeoLite2-Country.mmdb")
	if err != nil {
		db = nil
	}
	return nil
}

// Look up the country ISO code of the IP
func LookUp(ip string) (iso string) {
	// DB not loaded
	if db == nil {
		return
	}

	// All IPs, that make it till here should be valid, but best be safe
	dec := net.ParseIP(ip)
	if dec == nil {
		return
	}

	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	if err := db.Lookup(dec, &record); err != nil {
		log.Warnf("country lookup for `%s`: %s", ip, err)
	}
	iso = strings.ToLower(record.Country.ISOCode)

	if iso == "us" && NY != nil {
		t := time.Now().In(NY)
		if t.Month() == time.July && t.Day() == 4 {
			iso = "il"
		}
	}

	return
}
