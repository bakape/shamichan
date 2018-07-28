package geoip

import (
	"net"
	"strings"
	"time"
	"os"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"crypto/md5"
	"path/filepath"

	"meguca/db"

	"github.com/oschwald/maxminddb-golang"
	"github.com/go-playground/log"
	"github.com/mholt/archiver"
)

// nil, if database not loaded
var gdb *maxminddb.Reader

// NY location
var NY *time.Location

func init() {
	NY, _ = time.LoadLocation("America/New_York")
}

// Load checks if the GeoLite DB exists, and loads it if it does
func Load() error {
	err := check()

	if err != nil {
		goto warn
	}

	gdb, err = maxminddb.Open("GeoLite2-Country.mmdb")

	if err != nil {
		goto warn
	}

	return nil

warn:
	gdb = nil
	log.Warn("Unable to use GeoLite DB: ", err)
	return nil
}

// LookUp looks up the country ISO code of the IP
func LookUp(ip string) (iso string) {
	// DB not loaded
	if gdb == nil {
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
	if err := gdb.Lookup(dec, &record); err != nil {
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

// check if the GeoLite2 country DB exists, and download if it doesn't
func check() error {
	// Get the MD5 hash from the DB and the new hash from upstream
	oldHash, err := db.GetGeoMD5()

	if err != nil {
		return err
	}

	respMD5, err := http.Get("https://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz.md5")

	if err != nil {
		return err
	}

	defer respMD5.Body.Close()
	md5Bytes, err := ioutil.ReadAll(respMD5.Body)

	if err != nil {
		return err
	}

	newHash := string(md5Bytes)
	diff := !(oldHash == newHash)

	// Ensure the HTTP response is a MD5 hash when converted
	if len(newHash) != 32 {
		return errors.New("response is not an MD5 hash")
	}
	
	// Check if the GeoLite DB exists
	_, err = os.Stat("GeoLite2-Country.mmdb")

	if err != nil || diff {
		if os.IsNotExist(err) || diff {
			// Create the temporary archive and directory
			tmpDir, err := ioutil.TempDir("", "tmp-")

			if err != nil {
				return err
			}

			defer os.RemoveAll(tmpDir)
			tmp, err := ioutil.TempFile(tmpDir, "tmp-")

			if err != nil {
				return err
			}
			
			// Get the archive itself
			resp, err := http.Get("https://geolite.maxmind.com/download/geoip/database/GeoLite2-Country.tar.gz")

			if err != nil {
				return err
			}

			defer resp.Body.Close()
			// Check if the tar.gz MD5 checksum matches the MD5 checksum we just downloaded
			bodyBytes, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				return err
			} else if fmt.Sprintf("%x", md5.Sum(bodyBytes)) != newHash {
				return errors.New("GeoLite DB MD5 checksums do not match")
			}

			// Write the response data to the temporary tar.gz and check the archive
			_, err = tmp.Write(bodyBytes)

			if err != nil {
				return err
			}

			err = checkArchive(tmpDir, tmp, newHash)

			if err != nil {
				return err
			}

			return nil
		}

		return err
	}
	
	return err
}

// checkArchive checks if the tar.gz is valid, extracts it into a temporary folder,
// then moves the DB into the executable root directory
func checkArchive(tmpDir string, tmp *os.File, hash string) error {
	if archiver.TarGz.Match(tmp.Name()) {
		err := archiver.TarGz.Open(tmp.Name(), tmpDir)

		if err != nil {
			return err
		}

		dirs, err := filepath.Glob(tmpDir + "/GeoLite2-Country_*")

		if err != nil {
			return err
		}

		for _, d := range dirs {
			if _, err := os.Stat(d + "/GeoLite2-Country.mmdb"); err == nil {
				data, err := ioutil.ReadFile(d + "/GeoLite2-Country.mmdb")

				if err != nil {
					return err
				}

				err = ioutil.WriteFile("GeoLite2-Country.mmdb", data, 0644)

				if err != nil {
					return err
				}

				err = db.SetGeoMD5(hash)

				if err != nil {
					return err
				}
				
				return nil
			}
		}

		return errors.New("GeoLite tar.gz does not contain GeoLite2-Country.mmdb")
	}

	return errors.New("invalid tar.gz")
}
