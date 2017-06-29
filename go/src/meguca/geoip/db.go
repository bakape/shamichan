package geoip

import (
	"log"
	"net"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// nil, if database not loaded
var db *maxminddb.Reader

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
		log.Printf("country lookup for `%s`: %s", ip, err)
	}
	iso = record.Country.ISOCode

	if iso == "US" {
		t := time.Now()
		if t.Month() == time.July {
			day := t.Day()
			if 3 >= day && day <= 5 {
				iso = "IL"
			}
		}
	}

	return
}
