package server

import (
	"database/sql"
	"net/http"

	"github.com/Chiiruno/meguca/db"

	"github.com/badoux/goscraper"
)

// Get BitChute video title by ID
func bitChuteTitle(w http.ResponseWriter, r *http.Request) {
	httpError(w, r, func() (err error) {
		var title string
		id := extractParam(r, "id")

		title, err = db.GetBitChuteTitle(id)
		switch err {
		case nil:
		case sql.ErrNoRows:
			var doc *goscraper.Document
			doc, err = goscraper.Scrape(
				"https://www.bitchute.com/embed/"+id,
				3,
			)
			if err != nil {
				return
			}
			title = doc.Preview.Description
			err = db.WriteBitChuteTitle(id, title)
			if err != nil {
				return
			}
		default:
			return
		}

		w.Write([]byte(title))
		return
	}())
}
