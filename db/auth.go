package db

import (
	"net"
)

// GetIP returns an IP of the poster that created a post. Posts older than 7
// days will not have this information.
func GetPostIP(id uint64) (net.IP, error) {
	var s *string
	err := db.
		QueryRow(
			`select ip
			from posts
			where id = $1`,
			id,
		).
		Scan(&s)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return net.ParseIP(*s), nil
}
