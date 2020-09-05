package geoip

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/go-playground/log"
	"github.com/oschwald/geoip2-golang"
)

const denpa = "denpa"

var (
	once     sync.Once      // Ensure we only try to load the database once
	gdb      *geoip2.Reader // db-ip database
	stateMap = map[string]string{
		"Alabama":              "al",
		"Alaska":               "ak",
		"Arizona":              "az",
		"Arkansas":             "ar",
		"California":           "ca",
		"Colorado":             "co",
		"Connecticut":          "ct",
		"Delaware":             "de",
		"District of Columbia": "dc",
		"Florida":              "fl",
		"Georgia":              "ga",
		"Hawaii":               "hi",
		"Idaho":                "id",
		"Illinois":             "il",
		"Indiana":              "in",
		"Iowa":                 "ia",
		"Kansas":               "ks",
		"Kentucky":             "ky",
		"Louisiana":            "la",
		"Maine":                "me",
		"Maryland":             "md",
		"Massachusetts":        "ma",
		"Michigan":             "mi",
		"Minnesota":            "mn",
		"Mississippi":          "ms",
		"Missouri":             "mo",
		"Montana":              "mt",
		"Nebraska":             "ne",
		"Nevada":               "nv",
		"New Hampshire":        "nh",
		"New Jersey":           "nj",
		"New Mexico":           "nm",
		"New York":             "ny",
		"North Carolina":       "nc",
		"North Dakota":         "nd",
		"Ohio":                 "oh", // OHAAAYOOOUUUUU
		"Oklahoma":             "ok",
		"Oregon":               "or",
		"Pennsylvania":         "pa",
		"Rhode Island":         "ri",
		"South Carolina":       "sc",
		"South Dakota":         "sd",
		"Tennessee":            "tn",
		"Texas":                "tx",
		"Utah":                 "ut",
		"Vermont":              "vt",
		"Virginia":             "va",
		"Washington":           "wa",
		"West Virginia":        "wv",
		"Wisconsin":            "wi",
		"Wyoming":              "wy",
	}
)

// LookUp looks up the country ISO code of the IP
func LookUp(ip string) (country string) {
	once.Do(func() {
		var err error
		gdb, err = geoip2.Open("dbip-city-lite.mmdb")
		if err != nil {
			log.Warnf("geoip: could not load database: %s", err)
		}
	})

	dec := net.ParseIP(ip)
	if dec == nil || gdb == nil {
		// All IPs, that make it here should be valid, but best to be safe
		return denpa
	}

	record, err := gdb.City(dec)
	if err != nil {
		log.Warnf("geoip: could not lookup %s's country code: %s", ip, err)
		return denpa
	}

	country = strings.ToLower(record.Country.IsoCode)
	if len(country) < 2 {
		log.Warnf("could not lookup country for %s: %s", ip, country)
		return denpa
	}

	// Keep things safe, theoretically always just one result
	if country == "us" && len(record.Subdivisions) >= 1 {
		for _, v := range record.Subdivisions {
			state := v.Names["en"]
			if state != "" {
				return fmt.Sprintf("%s-%s", country, stateMap[state])
			}

			log.Warnf("could not lookup state for %s: %s", ip, country)
			return denpa
		}
	}

	return country
}
