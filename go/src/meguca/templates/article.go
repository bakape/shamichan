package templates

import (
	"html"
	"math"
	"strconv"
	"strings"
	"time"

	"meguca/auth"
	"meguca/common"
	"meguca/lang"
)

// GetPostModLog forwards db.GetPostModLog to avoid cyclic imports in cache
var GetPostModLog func(id uint64) []auth.ModLogEntry

// Extra data passed, when rendering an article
type articleContext struct {
	index, sticky, locked, rbText, pyu bool
	omit, imageOmit       int
	op                    uint64
	board, subject, root  string
	backlinks             backlinks
}

// Map of all backlinks on a page
type backlinks map[uint64]map[uint64]common.Link

// Returns image name with proper extension
func imageName(fileType uint8, name string) string {
	ext := common.Extensions[fileType]
	l := len(name)
	buf := make([]byte, l, l+1+len(ext))
	copy(buf, name)
	buf = append(buf, '.')
	return html.EscapeString(string(append(buf, ext...)))
}

// Renders the post creation time field
func formatTime(sec int64) string {
	ln := lang.Get().Common.Time

	t := time.Unix(sec, 0)
	year, m, day := t.Date()
	weekday := ln["week"][int(t.Weekday())]
	// Months are 1-indexed for some fucking reason
	month := ln["calendar"][int(m)-1]

	// Premature optimization
	buf := make([]byte, 0, 20+len(weekday)+len(month))
	buf = pad(buf, day)
	buf = append(buf, ' ')
	buf = append(buf, month...)
	buf = append(buf, ' ')
	buf = append(buf, strconv.Itoa(year)...)
	buf = append(buf, " ("...)
	buf = append(buf, weekday...)
	buf = append(buf, ") "...)
	buf = pad(buf, t.Hour())
	buf = append(buf, ':')
	buf = pad(buf, t.Minute())
	buf = append(buf, ':')
	buf = pad(buf, t.Second())

	return string(buf)
}

// Stringify an int and left-pad to at least double digits
func pad(buf []byte, i int) []byte {
	if i < 10 {
		buf = append(buf, '0')
	}
	return append(buf, strconv.Itoa(i)...)
}

// Formats a human-readable representation of file size
func readableFileSize(s int) string {
	format := func(n, end string) string {
		l := len(n)
		buf := make([]byte, l, l+len(end))
		copy(buf, n)
		return string(append(buf, end...))
	}

	switch {
	case s < (1 << 10):
		return format(strconv.Itoa(s), " B")
	case s < (1 << 20):
		return format(strconv.Itoa(s/(1<<10)), " KB")
	default:
		n := strconv.FormatFloat(float64(s)/(1<<20), 'f', 1, 32)
		return format(n, " MB")
	}
}

// Extract reverse links to linked posts on a page
func extractBacklinks(cap int, threads ...common.Thread) backlinks {
	bls := make(backlinks, cap)
	register := func(p common.Post, op uint64, board string) {
		for _, l := range p.Links {
			m, ok := bls[l.ID]
			if !ok {
				m = make(map[uint64]common.Link, 4)
				bls[l.ID] = m
			}
			m[p.ID] = common.Link{
				ID:    p.ID,
				OP:    op,
				Board: board,
			}
		}
	}

	for _, t := range threads {
		register(t.Post, t.ID, t.Board)
		for _, p := range t.Posts {
			register(p, t.ID, t.Board)
		}
	}

	return bls
}

// Returns 3 mod-log related messages by post ID
func parseModLog(p common.Post) (any bool, attrs string, msgs string) {
	ln := lang.Get().Common.Posts
	log := GetPostModLog(p.ID)

	for i, val := range [3]bool{p.Banned, p.Deleted, p.MeidoVision} {
		if val {
			switch i {
			case 0:
				any = true
				attrs = " banned"
				msgs = ln["banned"]

				if log[i].ID != 0 {
					msgs += ` BY "` + log[i].By + `" FOR ` +
						strings.ToUpper(secondsToTime(float64(log[i].Length))) +
						": " + log[i].Reason
				}
			case 1:
				if log[i].ID != 0 {
					if (any) {
						msgs += "<br>"
					}
		
					any = true
					attrs += " deleted"
					msgs += ln["deleted"] + ` BY "` + log[i].By + `"`
				}
			case 2:
				if (any) {
					msgs += "<br>"
				}

				any = true
				attrs += " meido-vision"
				msgs += ln["meidoVision"]
				
				if log[i].ID != 0 {
					msgs += ` BY "` + log[i].By + `"`
				}
			}
		}
	}

	return
}

// Returns human readable time
func secondsToTime(s float64) string {
	time := math.Floor(s) / 60
	divide := [4]float64{60, 24, 30, 12}
	unit := [4]string{"minute", "hour", "day", "month"}

	for i := 0; i < len(divide); i++ {
		if time < divide[i] {
			return pluralize(int(ToFixed(time, 0)), unit[i])
		}

		time = math.Floor(time / divide[i])
	}

	return pluralize(int(ToFixed(time, 0)), "year")
}

// ToFixed truncates a float64
func ToFixed(num float64, precision int) float64 {
    out := math.Pow(10, float64(precision))
    return float64(round(num * out)) / out
}

func round(num float64) int {
    return int(num + math.Copysign(0.5, num))
}

// Returns the stringified n + the plural or singular word
// from the language by index word
func pluralize(n int, word string) string {
	ln := lang.Get().Common.Plurals[word]
	b := make([]byte, 0, 32)
	b = strconv.AppendInt(b, int64(n), 10)
	b = append(b, ' ')

	switch n {
	case 1, -1:
		b = append(b, ln[0]...)
	default:
		b = append(b, ln[1]...)
	}

	return string(b)
}
