package htmlcheck

import (
	"sort"

	"github.com/axllent/mailpit/internal/tools"
)

// Platforms returns all platforms with their respective email clients
func Platforms() (map[string][]string, error) {
	// [platform]clients
	data := make(map[string][]string)

	if err := loadJSONData(); err != nil {
		return data, err
	}

	for _, t := range cie.Data {
		for family, stats := range t.Stats {
			niceFamily := cie.NiceNames.Family[family]
			for platform := range stats.(map[string]interface{}) {
				c, found := data[platform]
				if !found {
					data[platform] = []string{}
				}
				if !tools.InArray(niceFamily, c) {
					c = append(c, niceFamily)
					data[platform] = c
				}
			}
		}
	}

	for group, clients := range data {
		sort.Slice(clients, func(i, j int) bool {
			return clients[i] < clients[j]
		})
		data[group] = clients
	}

	return data, nil
}
