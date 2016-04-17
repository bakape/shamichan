// Generates, persists and compares image perceptual hashes for duplicate
// detection.

package imager

import (
	"github.com/bakape/meguca/config"
	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
	"github.com/jteeuwen/imghash"
	"log"
	"time"
)

var (
	// Request channel for image deduplication
	dedupImage = make(chan dedupRequest)

	// Interval at which to clean up expired image hash entries. Overriden in
	// tests.
	cleanUpInterval = time.Minute
)

// Request to verify image has no duplicates and persist it to the stored image
// hash set
type dedupRequest struct {
	entry HashEntry

	// Channel to receive the post number, that contains a mactching image, or
	// 0, if no matches found
	res chan uint64
}

// HashEntry is a storage structs of a single post's image's hash
type HashEntry struct {
	// Parent post number
	ID uint64 `gorethink:"id"`

	// Perceptual hash of the image
	Hash uint64 `gorethink:"hash"`
}

// DatabaseHashEntry includes the additional expires field, that only exists
// database-side
type DatabaseHashEntry struct {
	HashEntry
	Expires r.Term `gorethink:"expires"`
}

// InitImager cleans up any dangling artefacts and start the processing
// goroutines. Needs to be called after a database connection is established.
func InitImager() error {
	go handlePerceptualHashes(nil)

	// Clean up on server start
	return cleanUpHashes()
}

// Handles dudplication and persistance of perceptual hashes as a dedicated
// goroutine. The close channel is only intended for testing, as the goroutine
// never stops during production.
func handlePerceptualHashes(close <-chan struct{}) {
	// Timer for cleaning up old entries from the database
	cleanUp := time.Tick(cleanUpInterval)

	for {
		select {
		case req := <-dedupImage:
			handleDedupRequest(req)
		case <-cleanUp:
			if err := cleanUpHashes(); err != nil {
				log.Println(err)
			}
		case <-close:
			return
		}
	}
}

func handleDedupRequest(req dedupRequest) {
	// Retrive all entries from the DB
	var entries []HashEntry
	err := db.DB(db.GetMain("imageHashes").Field("hashes")).All(&entries)
	if err != nil {
		req.res <- 0
		log.Printf("Error retrieving image hashes: %s\n", err)
		return
	}

	minDistance := uint64(config.Images().DuplicateThreshold)
	var matched bool
	for _, entry := range entries {
		if imghash.Distance(req.entry.Hash, entry.Hash) <= minDistance {
			req.res <- entry.ID
			matched = true
			break
		}
	}

	if !matched {
		if err := persistHash(req.entry); err != nil {
			log.Printf("Error persisting image hash: %s\n", err)
		}
		req.res <- 0
	}
}

// Persist hash entry to the database
func persistHash(entry HashEntry) error {
	update := map[string]r.Term{
		"hashes": r.Row.Field("hashes").Append(DatabaseHashEntry{
			HashEntry: entry,
			Expires:   r.Now(),
		}),
	}
	return db.DB(db.GetMain("imageHashes").Update(update)).Exec()
}

// Remove expired image hashes
func cleanUpHashes() (err error) {
	query := r.
		Table("main").
		Get("imageHashes").
		Update(map[string]r.Term{
			"hashes": r.Row.Field("hashes").Filter(func(doc r.Term) r.Term {
				return doc.Field("expires").
					Gt(r.Now().Sub(config.Images().DulicateLifetime))
			}),
		})
	err = db.DB(query).Exec()
	if err != nil {
		err = util.WrapError("Error cleaning up image hashes", err)
	}
	return
}
