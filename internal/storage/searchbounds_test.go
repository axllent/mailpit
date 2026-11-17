package storage

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
)

func searchTerms(n int) string {
	return strings.TrimSpace(strings.Repeat("common ", n))
}

// A search with a huge number of terms made the query builder allocate about 500 MiB (for 1 MB of terms) before
// SQLite refused it. Too many terms is an error, for every kind of search: for the ones which change
// messages it is also the safe thing to do, as ignoring terms would change more than was asked for.
func TestSearchTermsAreLimited(t *testing.T) {
	setup("")
	_ = DeleteAllMessages()
	seedMessages(t, 20, false)

	tooMany := searchTerms(MaxSearchTerms + 1)
	atLimit := searchTerms(MaxSearchTerms)

	t.Run("control: the maximum number of terms works", func(t *testing.T) {
		if _, total, err := Search(atLimit, "", 0, 0, 50); err != nil || total != 20 {
			t.Errorf("total %d, error %v", total, err)
		}
		if n, err := SearchUnreadCount(atLimit, "", 0); err != nil || n != 20 {
			t.Errorf("unread %d, error %v", n, err)
		}
	})

	t.Run("quoted phrases are one term", func(t *testing.T) {
		phrase := strings.Repeat(`"common bulk0000001 " `, 3) + searchTerms(MaxSearchTerms-3)
		if _, _, err := Search(phrase, "", 0, 0, 50); err != nil {
			t.Errorf("error %v", err)
		}
	})

	t.Run("exploit: too many terms", func(t *testing.T) {
		if _, _, err := Search(tooMany, "", 0, 0, 50); err == nil {
			t.Error("Search: no error")
		}
		if _, err := SearchUnreadCount(tooMany, "", 0); err == nil {
			t.Error("SearchUnreadCount: no error")
		}
		if err := SetSearchReadStatus(tooMany, "", true); err == nil {
			t.Error("SetSearchReadStatus: no error")
		}
		if err := DeleteSearch(tooMany, ""); err == nil {
			t.Error("DeleteSearch: no error")
		}
		if got := CountTotal(); got != 20 {
			t.Errorf("DeleteSearch with too many terms deleted messages: %d left", got)
		}
		if got := CountUnread(); got != 20 {
			t.Errorf("SetSearchReadStatus with too many terms changed messages: %d unread", got)
		}
	})

	t.Run("exploit: a very long query", func(t *testing.T) {
		if _, _, err := Search(strings.Repeat("a ", 500000), "", 0, 0, 50); err == nil {
			t.Error("no error")
		}
	})

	t.Run("a tag filter with too many terms is ignored", func(t *testing.T) {
		orig := config.TagFilters
		t.Cleanup(func() { config.TagFilters = orig; LoadTagFilters() })

		// the type of the filters is not exported, so they are built with reflection
		filters := reflect.MakeSlice(reflect.TypeOf(config.TagFilters), 0, 2)
		for _, f := range []struct{ match, tag string }{{tooMany, "toomany"}, {atLimit, "fine"}} {
			filter := reflect.New(filters.Type().Elem()).Elem()
			filter.FieldByName("Match").SetString(f.match)
			filter.FieldByName("Tags").Set(reflect.ValueOf([]string{f.tag}))
			filters = reflect.Append(filters, filter)
		}
		reflect.ValueOf(&config.TagFilters).Elem().Set(filters)
		LoadTagFilters()

		if len(tagFilters) != 1 || tagFilters[0].Tags[0] != "fine" {
			t.Errorf("tag filters: %+v", tagFilters)
		}
	})
}

// Only the messages of the page are kept, but the total is every match.
func TestSearchPages(t *testing.T) {
	setup("")
	_ = DeleteAllMessages()
	ids := seedMessages(t, 120, false)

	// newest first, which is the order the messages were seeded in
	all, total, err := Search("common", "", 0, 0, 1000)
	if err != nil || total != 120 || len(all) != 120 {
		t.Fatalf("all: %d results, total %d, error %v", len(all), total, err)
	}
	for i, m := range all {
		if m.ID != ids[i] {
			t.Fatalf("result %d is %s, want %s", i, m.ID, ids[i])
		}
	}

	for _, tt := range []struct {
		name         string
		start, limit int
		want         []string
	}{
		{"first page", 0, 50, ids[0:50]},
		{"second page", 50, 50, ids[50:100]},
		{"short last page", 100, 50, ids[100:120]},
		{"inside", 7, 3, ids[7:10]},
		{"start at the total", 120, 50, nil},
		{"start after the total", 500, 50, nil},
		{"no limit", 0, 0, nil},
		{"limit larger than the total", 0, math.MaxInt, ids},
		{"start and a huge limit", 119, math.MaxInt, ids[119:]},
		{"negative start is the first page", -5, 2, ids[0:2]},
		{"negative limit is the default of 50", 0, -1, ids[0:50]},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := Search("common", "", tt.start, 0, tt.limit)
			if err != nil {
				t.Fatal(err)
			}
			if total != 120 {
				t.Errorf("total %d, want 120", total)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("%d results, want %d", len(got), len(tt.want))
			}
			for i, m := range got {
				if m.ID != tt.want[i] {
					t.Errorf("result %d is %s, want %s", i, m.ID, tt.want[i])
				}
			}
			if got == nil {
				t.Error("results are nil, not an empty list")
			}
		})
	}

	// control: a search which matches few
	few, total, err := Search(fmt.Sprintf("common %s", ids[5]), "", 0, 0, 50)
	if err != nil || total != 1 || len(few) != 1 || few[0].ID != ids[5] {
		t.Errorf("one match: %d results, total %d, error %v", len(few), total, err)
	}
}

// A size which is larger than a database can compare with (more than MaxInt64 bytes) made the search fail
// with "uint64 values with high bit set are not supported". It is the largest size instead, which is more
// than any message.
func TestSearchSizeIsClamped(t *testing.T) {
	huge := strings.Repeat("9", 400)

	for in, want := range map[string]uint64{
		// controls: sizes which are not too large are as they were
		"50":    50,
		"1k":    1024,
		"1.5mb": 1572864,
		// exploit: sizes which are
		"9223372036854775807":     math.MaxInt64,
		"9223372036854775808":     math.MaxInt64,
		"99999999999999999999":    math.MaxInt64,
		"9999999999999999999999k": math.MaxInt64,
		"99999999999999999999mb":  math.MaxInt64,
		huge:                      math.MaxInt64, // beyond a float64, which ParseFloat reports as out of range
		huge + "mb":               math.MaxInt64,
		// not sizes
		"1e30": 0,
		"12x":  0,
	} {
		name := in
		if len(name) > 30 {
			name = fmt.Sprintf("%d digits", len(in))
		}
		if got := sizeToBytes(in); got != want {
			t.Errorf("%s: %d, want %d", name, got, want)
		}
	}

	setup("")
	_ = DeleteAllMessages()
	seedMessages(t, 20, false) // 100 bytes each

	for _, tt := range []struct {
		search string
		want   int
	}{
		{"larger:50", 20},
		{"larger:200", 0},
		{"smaller:200", 20},
		{"smaller:50", 0},
		{"larger:99999999999999999999", 0},
		{"smaller:99999999999999999999", 20},
		{"larger:" + strings.Repeat("9", 400), 0},
		{"smaller:" + strings.Repeat("9", 400) + "mb", 20},
	} {
		_, total, err := Search(tt.search, "", 0, 0, 50)
		if err != nil || total != tt.want {
			t.Errorf("%.40q: %d results, error %v, want %d", tt.search, total, err, tt.want)
		}
		if _, err := SearchUnreadCount(tt.search, "", 0); err != nil {
			t.Errorf("%.40q: unread count: %v", tt.search, err)
		}
	}
}
