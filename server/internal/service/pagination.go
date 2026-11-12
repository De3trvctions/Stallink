package service

import (
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func pageParams(r *http.Request) (page, limit int, search string, ok bool) {
	page, limit = 1, 20
	var err error
	if s := r.URL.Query().Get("page"); s != "" {
		page, err = strconv.Atoi(s)
		if err != nil || page < 1 || page > 100000 {
			return 0, 0, "", false
		}
	}
	if s := r.URL.Query().Get("limit"); s != "" {
		limit, err = strconv.Atoi(s)
		if err != nil || limit < 1 || limit > 50 {
			return 0, 0, "", false
		}
	}
	search = strings.TrimSpace(r.URL.Query().Get("q"))
	if !utf8.ValidString(search) || utf8.RuneCountInString(search) > 100 {
		return 0, 0, "", false
	}
	return page, limit, search, true
}
