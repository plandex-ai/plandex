package db

import "github.com/lib/pq"

func IsNonUniqueErr(err error) bool {
	if err, ok := err.(*pq.Error); ok {
		if err.Code == "23505" {
			return true
		}
	}
	return false
}
