package storage

import (
	"regexp"
	"strings"

	"github.com/leporo/sqlf"
)

// SearchParser returns the SQL syntax for the database search based on the search arguments
func searchParser(args []string) *sqlf.Stmt {
	q := sqlf.From("mailbox").
		Select(`ID, Data, read, 
			json_extract(Data, '$.To') as ToJSON, 
			json_extract(Data, '$.From') as FromJSON, 
			json_extract(Data, '$.Subject') as Subject, 
			json_extract(Data, '$.Attachments') as Attachments
		`).
		OrderBy("Sort DESC").
		Limit(200)

	for _, w := range args {
		if cleanString(w) == "" {
			continue
		}

		exclude := false
		// search terms starting with a `-` or `!` imply an exclude
		if len(w) > 1 && (strings.HasPrefix(w, "-") || strings.HasPrefix(w, "!")) {
			exclude = true
			w = w[1:]
		}

		re := regexp.MustCompile(`[a-zA-Z0-9]+`)
		if !re.MatchString(w) {
			continue
		}

		if strings.HasPrefix(w, "to:") {
			w = cleanString(w[3:])
			if w != "" {
				if exclude {
					q.Where("ToJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("ToJSON LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "from:") {
			w = cleanString(w[5:])
			if w != "" {
				if exclude {
					q.Where("FromJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("FromJSON LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "subject:") {
			w = cleanString(w[8:])
			if w != "" {
				if exclude {
					q.Where("Subject NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("Subject LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if w == "is:read" {
			if exclude {
				q.Where("Read = 0")
			} else {
				q.Where("Read = 1")
			}
		} else if w == "is:unread" {
			if exclude {
				q.Where("Read = 1")
			} else {
				q.Where("Read = 0")
			}
		} else if w == "has:attachment" || w == "has:attachments" {
			if exclude {
				q.Where("Attachments = 0")
			} else {
				q.Where("Attachments > 0")
			}
		} else {
			// search text
			if exclude {
				q.Where("search NOT LIKE ?", "%"+cleanString(escPercentChar(w))+"%")
			} else {
				q.Where("search LIKE ?", "%"+cleanString(escPercentChar(w))+"%")
			}
		}
	}

	return q
}
