package geoip

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/abh/geoip"
	"github.com/go-playground/log"
)

var (
	once  sync.Once    // Ensure we only try to load the database once
	gdbV4 *geoip.GeoIP // GeoIP database for IPv4
	gdbV6 *geoip.GeoIP // GeoIP database for IPv6
)

// NY location
var NY *time.Location

func init() {
	NY, _ = time.LoadLocation("America/New_York")
}

// LookUp looks up the country ISO code of the IP
func LookUp(ip string) (iso string) {
	once.Do(func() {
		open := func(path string) (db *geoip.GeoIP) {
			db, err := geoip.Open(path)
			if err != nil {
				log.Warnf("geoip: could not load database %s: %s", path, err)
			}
			return
		}

		gdbV4 = open("/usr/share/GeoIP/GeoIP.dat")
		gdbV6 = open("/usr/share/GeoIP/GeoIPv6.dat")
	})

	dec := net.ParseIP(ip)
	switch {
	case dec == nil:
		// All IPs, that make it till here should be valid, but best be safe
		return
	case dec.To4() != nil:
		if gdbV4 == nil {
			return
		}
		iso, _ = gdbV4.GetCountry(ip)
	default:
		if gdbV6 == nil {
			return
		}
		iso, _ = gdbV6.GetCountry_v6(ip)
	}

	// Error returned
	if len(iso) != 2 {
		log.Warnf("could not lookup country for %s: %s", ip, iso)
		return ""
	} else {
		iso = strings.ToLower(iso)
	}

	if iso == "us" && NY != nil {
		t := time.Now().In(NY)
		if t.Month() == time.July && t.Day() == 4 {
			iso = "il"
		}
	}

	return
}
