package storage

import (
	"regexp"
	"strings"

	"github.com/leporo/sqlf"
)

// SearchParser returns the SQL syntax for the database search based on the search arguments
func searchParser(args []string) *sqlf.Stmt {
	q := sqlf.From("mailbox").
		Select(`Created, ID, MessageID, Subject, Metadata, Size, Attachments, Read, Tags,
			IFNULL(json_extract(Metadata, '$.To'), '{}') as ToJSON,
			IFNULL(json_extract(Metadata, '$.From'), '{}') as FromJSON,
			IFNULL(json_extract(Metadata, '$.Cc'), '{}') as CcJSON,
			IFNULL(json_extract(Metadata, '$.Bcc'), '{}') as BccJSON
		`).OrderBy("Created DESC")

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
		} else if strings.HasPrefix(w, "cc:") {
			w = cleanString(w[3:])
			if w != "" {
				if exclude {
					q.Where("CcJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("CcJSON LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "bcc:") {
			w = cleanString(w[4:])
			if w != "" {
				if exclude {
					q.Where("BccJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("BccJSON LIKE ?", "%"+escPercentChar(w)+"%")
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
		} else if strings.HasPrefix(w, "message-id:") {
			w = cleanString(w[11:])
			if w != "" {
				if exclude {
					q.Where("MessageID NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("MessageID LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "tag:") {
			w = cleanString(w[4:])
			if w != "" {
				if exclude {
					q.Where("Tags NOT LIKE ?", "%\""+escPercentChar(w)+"\"%")
				} else {
					q.Where("Tags LIKE ?", "%\""+escPercentChar(w)+"\"%")
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
				q.Where("SearchText NOT LIKE ?", "%"+cleanString(escPercentChar(w))+"%")
			} else {
				q.Where("SearchText LIKE ?", "%"+cleanString(escPercentChar(w))+"%")
			}
		}
	}

	return q
}
