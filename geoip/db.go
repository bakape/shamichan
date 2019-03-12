package geoip

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"github.com/bakape/meguca/db"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/log"
	"github.com/oschwald/maxminddb-golang"

	// TODO: Rewrite for archiver v3
	"gopkg.in/mholt/archiver.v2"
)

// nil, if database not loaded
var (
	// Ensure we only load the old GeoIP DB once on server start
	once sync.Once
	// Ensures no data races
	rw sync.RWMutex
	// GeoIP database
	gdb *maxminddb.Reader
)

// NY location
var NY *time.Location

func init() {
	NY, _ = time.LoadLocation("America/New_York")
}

// Load checks if the GeoLite DB exists, and calls load it if it does
func Load() error {
	go func() {
		if err := check(); err != nil {
			rw.Lock()
			defer rw.Unlock()
			gdb = nil
			log.Warn("Unable to use GeoLite DB: ", err)
		}
	}()

	return nil
}

// loads the GeoLite DB
func load() (err error) {
	rw.Lock()
	defer rw.Unlock()

	if gdb != nil {
		err := gdb.Close()

		if err != nil {
			return err
		}

		gdb = nil
	}

	gdb, err = maxminddb.Open("GeoLite2-Country.mmdb")
	return
}

// LookUp looks up the country ISO code of the IP
func LookUp(ip string) (iso string) {
	rw.RLock()
	defer rw.RUnlock()

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

	// Ensure the HTTP response is a MD5 hash when converted
	if len(newHash) != 32 {
		return errors.New("response is not an MD5 hash")
	}

	// Load the old DB one time on server start, if applicable before checking the MD5
	_, err = os.Stat("GeoLite2-Country.mmdb")
	invalid := os.IsNotExist(err)

	once.Do(func() {
		if !invalid {
			err = load()
		}
	})

	// Check if the GeoLite DB exists
	if err != nil || invalid || oldHash != newHash {
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
	}

	return load()
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

				return db.SetGeoMD5(hash)
			}
		}

		return errors.New("GeoLite tar.gz does not contain GeoLite2-Country.mmdb")
	}

	return errors.New("invalid tar.gz")
}
