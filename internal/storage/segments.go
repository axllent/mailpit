package storage

import (
	"fmt"
	"strings"

	"github.com/axllent/mailpit/internal/segment"
	"github.com/leporo/sqlf"
)

// segmentFiltered returns whether queries for a segment must be filtered.
// Nothing is filtered when segments are disabled, so all messages remain visible
// regardless of the segment they were stored in.
func segmentFiltered(seg string) bool {
	return segment.Enabled() && seg != segment.All
}

// whereSegment limits a query on the mailbox table to a segment.
// The column is the (optionally aliased) Segment column, eg: "m.Segment".
func whereSegment(q *sqlf.Stmt, column, seg string) *sqlf.Stmt {
	if !segmentFiltered(seg) {
		return q
	}

	return q.Where(column+" = ?", seg)
}

// segmentSQL returns an SQL condition (prefixed with AND) and its arguments to
// limit a raw query on the mailbox table to a segment.
func segmentSQL(column, seg string) (string, []any) {
	if !segmentFiltered(seg) {
		return "", nil
	}

	return " AND " + column + " = ?", []any{seg}
}

// messageInSegment returns whether a message exists within a segment
func messageInSegment(seg, id string) bool {
	ids, err := idsInSegment(seg, []string{id})

	return err == nil && len(ids) == 1
}

// idsInSegment returns the subset of message IDs which belong to the segment.
// When segments are not filtered the IDs are returned as-is.
func idsInSegment(seg string, ids []string) ([]string, error) {
	if !segmentFiltered(seg) || len(ids) == 0 {
		return ids, nil
	}

	result := []string{}

	for _, chunk := range chunkBy(ids, 1000) {
		args := make([]any, 0, len(chunk)+1)
		args = append(args, seg)
		for _, id := range chunk {
			args = append(args, id)
		}

		query := fmt.Sprintf(`SELECT ID FROM %s WHERE Segment = ? AND ID IN (?%s)`, tenant("mailbox"), strings.Repeat(",?", len(chunk)-1)) // #nosec

		rows, err := db.Query(query, args...)
		if err != nil {
			return result, err
		}

		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return result, err
			}
			result = append(result, id)
		}

		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return result, err
		}
	}

	return result, nil
}
