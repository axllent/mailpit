package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/segment"
)

// storeInSegments stores one plain text and one MIME message in each of the
// default, "alpha" and "beta" segments, and returns the IDs per segment.
func storeInSegments(t *testing.T) map[string][]string {
	t.Helper()

	ids := map[string][]string{}

	for _, seg := range []string{segment.Default, "alpha", "beta"} {
		for _, msg := range [][]byte{testTextEmail, testMimeEmail} {
			b := append([]byte{}, msg...)
			id, err := Store(seg, &b, nil)
			if err != nil {
				t.Fatalf("error storing message in segment %q: %s", seg, err.Error())
			}
			ids[seg] = append(ids[seg], id)
		}
	}

	return ids
}

func enableSegments(t *testing.T) {
	t.Helper()
	config.SegmentDomain = "mail.example.com"
	t.Cleanup(func() { config.SegmentDomain = "" })
}

func TestSegmentsIsolation(t *testing.T) {
	setup("")
	defer Close()
	enableSegments(t)

	ids := storeInSegments(t)

	for seg, segIDs := range ids {
		other := "beta"
		if seg == "beta" {
			other = "alpha"
		}

		// list
		list, err := List(seg, 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, len(list), 2, "incorrect number of messages listed in segment "+seg)
		for _, m := range list {
			if !inSlice(m.ID, segIDs) {
				t.Fatalf("segment %q listed message %s from another segment", seg, m.ID)
			}
		}

		// search
		results, total, err := Search(seg, "from:sender@example.com", "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, total, 1, "incorrect number of search results in segment "+seg)
		if !inSlice(results[0].ID, segIDs) {
			t.Fatalf("segment %q search returned message %s from another segment", seg, results[0].ID)
		}

		// counts & stats
		assertEqual(t, CountTotal(seg), uint64(2), "incorrect total in segment "+seg)
		assertEqual(t, CountUnread(seg), uint64(2), "incorrect unread in segment "+seg)

		// messages of another segment cannot be accessed
		foreignID := ids[other][1]
		if _, err := GetMessage(seg, foreignID); err == nil {
			t.Fatalf("segment %q could read message %s of segment %q", seg, foreignID, other)
		}
		if _, err := GetMessageRaw(seg, foreignID); err == nil {
			t.Fatalf("segment %q could read raw message %s of segment %q", seg, foreignID, other)
		}
		if _, err := GetAttachmentPart(seg, foreignID, "2"); err == nil {
			t.Fatalf("segment %q could read an attachment of message %s of segment %q", seg, foreignID, other)
		}
		if _, err := GetMetadata(seg, foreignID); err == nil {
			t.Fatalf("segment %q could read metadata of message %s of segment %q", seg, foreignID, other)
		}
		if _, err := SetMessageTags(seg, foreignID, []string{"Hacked"}); err == nil {
			t.Fatalf("segment %q could tag message %s of segment %q", seg, foreignID, other)
		}

		// messages of the own segment can be accessed
		if _, err := GetMessage(seg, segIDs[1]); err != nil {
			t.Fatalf("segment %q could not read its own message: %s", seg, err.Error())
		}
		if _, err := GetAttachmentPart(seg, segIDs[1], "2"); err != nil {
			t.Fatalf("segment %q could not read its own attachment: %s", seg, err.Error())
		}
	}

	// unknown segments are empty
	assertEqual(t, CountTotal("gamma"), uint64(0), "unknown segment is not empty")

	// all segments, as used for global statistics
	assertEqual(t, CountTotal(segment.All), uint64(6), "incorrect total for all segments")

	// reading a message marks it as read, so reset first
	for _, seg := range []string{"alpha", "beta"} {
		if err := MarkAllUnread(seg); err != nil {
			t.Fatal(err)
		}
	}

	// mark read across segments only affects the own segment
	if err := MarkRead("alpha", append(ids["alpha"], ids["beta"]...)); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, CountUnread("alpha"), uint64(0), "alpha messages not marked read")
	assertEqual(t, CountUnread("beta"), uint64(2), "beta messages marked read by alpha")

	if err := MarkAllUnread("alpha"); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, CountUnread("alpha"), uint64(2), "alpha messages not marked unread")

	if err := MarkAllRead("beta"); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, CountUnread("beta"), uint64(0), "beta messages not marked read")
	assertEqual(t, CountUnread("alpha"), uint64(2), "alpha messages marked read by beta")

	// deleting messages of another segment does nothing
	if err := DeleteMessages("alpha", ids["beta"]); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, CountTotal("beta"), uint64(2), "alpha deleted messages of beta")

	// delete search only deletes in the own segment
	if err := DeleteSearch("alpha", "from:sender@example.com", ""); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, CountTotal("alpha"), uint64(1), "delete search did not delete in alpha")
	assertEqual(t, CountTotal("beta"), uint64(2), "delete search in alpha deleted messages of beta")

	// delete all only deletes the own segment
	if err := DeleteAllMessages("alpha"); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, CountTotal("alpha"), uint64(0), "delete all did not delete alpha")
	assertEqual(t, CountTotal("beta"), uint64(2), "delete all in alpha deleted messages of beta")
	assertEqual(t, CountTotal(segment.Default), uint64(2), "delete all in alpha deleted messages of the default segment")
}

func TestSegmentsDisabled(t *testing.T) {
	setup("")
	defer Close()

	// store messages in segments while enabled
	config.SegmentDomain = "mail.example.com"
	ids := storeInSegments(t)
	config.SegmentDomain = ""

	// with segments disabled every message is visible, whatever the segment
	assertEqual(t, CountTotal(segment.Default), uint64(6), "all messages should be visible when disabled")
	if _, err := GetMessage(segment.Default, ids["beta"][0]); err != nil {
		t.Fatalf("message in segment beta should be readable when disabled: %s", err.Error())
	}
}

func TestSegmentsDuplicateMessageIDs(t *testing.T) {
	setup("")
	defer Close()
	enableSegments(t)

	b := append([]byte{}, testTextEmail...)
	if _, err := Store("alpha", &b, nil); err != nil {
		t.Fatal(err)
	}

	list, err := List("alpha", 0, 0, 1)
	if err != nil || len(list) != 1 {
		t.Fatal("expected one message in alpha")
	}

	if !MessageIDExists("alpha", list[0].MessageID) {
		t.Fatal("expected Message-ID to exist in alpha")
	}
	if MessageIDExists("beta", list[0].MessageID) {
		t.Fatal("Message-ID of alpha should not exist in beta")
	}
}

func TestSegmentsTags(t *testing.T) {
	setup("")
	defer Close()
	enableSegments(t)

	ids := storeInSegments(t)

	for _, seg := range []string{"alpha", "beta"} {
		if _, err := SetMessageTags(seg, ids[seg][0], []string{"Shared", "Only " + seg}); err != nil {
			t.Fatal(err)
		}
	}

	alphaTags := GetAllTags("alpha")
	assertEqual(t, len(alphaTags), 2, "incorrect number of alpha tags")
	assertEqual(t, alphaTags[0], "Only alpha", "incorrect alpha tag")
	assertEqual(t, alphaTags[1], "Shared", "incorrect alpha tag")

	assertEqual(t, len(GetAllTags(segment.Default)), 0, "default segment should have no tags")
	assertEqual(t, len(GetAllTagsCount("beta")), 2, "incorrect number of beta tag counts")

	// tags of another segment cannot be renamed or deleted
	if err := RenameTag("alpha", "Only beta", "Renamed"); err == nil {
		t.Fatal("alpha could rename a tag only used by beta")
	}
	if err := DeleteTag("alpha", "Only beta"); err == nil {
		t.Fatal("alpha could delete a tag only used by beta")
	}

	// renaming a shared tag only affects the own segment
	if err := RenameTag("alpha", "Shared", "Alpha shared"); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, GetAllTags("alpha")[0], "Alpha shared", "shared tag not renamed in alpha")
	assertEqual(t, GetAllTags("beta")[1], "Shared", "shared tag renamed in beta")

	// renaming a tag only used in the own segment
	if err := RenameTag("beta", "Only beta", "Beta only"); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, GetAllTags("beta")[0], "Beta only", "tag not renamed in beta")

	// deleting a shared tag only affects the own segment
	if _, err := SetMessageTags("alpha", ids["alpha"][1], []string{"Shared"}); err != nil {
		t.Fatal(err)
	}
	if err := DeleteTag("beta", "Shared"); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, len(GetAllTags("beta")), 1, "shared tag not deleted in beta")
	if !inSlice("Shared", GetAllTags("alpha")) {
		t.Fatal("deleting a tag in beta removed it from alpha")
	}
}

func TestSegmentsMigration(t *testing.T) {
	origDatabase := config.Database
	defer func() { config.Database = origDatabase }()

	setup("")
	Close()

	// use a persistent database file so it can be reopened
	config.Database = filepath.Join(t.TempDir(), "mailpit.db")
	if err := InitDB(); err != nil {
		t.Fatal(err)
	}

	b := append([]byte{}, testTextEmail...)
	id, err := Store(segment.Default, &b, nil)
	if err != nil {
		t.Fatal(err)
	}

	// simulate a database created before segments existed
	for _, q := range []string{
		`DROP INDEX ` + tenant("idx_segment_created"),
		`ALTER TABLE ` + tenant("mailbox") + ` DROP COLUMN Segment`,
		`DELETE FROM ` + tenant("schemas") + ` WHERE Version = '1.34.1'`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("%s: %s", q, err.Error())
		}
	}

	var hasColumn int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = 'Segment'`, tenant("mailbox")).Scan(&hasColumn); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, hasColumn, 0, "Segment column was not dropped")

	Close()

	// reopening applies the migration
	if err := InitDB(); err != nil {
		t.Fatal(err)
	}
	defer Close()

	enableSegments(t)

	var seg sql.NullString
	if err := db.QueryRow(`SELECT Segment FROM `+tenant("mailbox")+` WHERE ID = ?`, id).Scan(&seg); err != nil {
		t.Fatal(err)
	}
	assertEqual(t, seg.String, segment.Default, "existing message not in the default segment")

	assertEqual(t, CountTotal(segment.Default), uint64(1), "existing message not visible in the default segment")
	assertEqual(t, CountTotal("alpha"), uint64(0), "existing message visible in another segment")
}

func TestSegmentsPruneMax(t *testing.T) {
	setup("")
	defer Close()
	enableSegments(t)

	for range 5 {
		b := append([]byte{}, testTextEmail...)
		if _, err := Store("alpha", &b, nil); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 {
		b := append([]byte{}, testTextEmail...)
		if _, err := Store("beta", &b, nil); err != nil {
			t.Fatal(err)
		}
	}

	config.MaxMessages = 3
	defer func() { config.MaxMessages = 0 }()

	pruneMessages()

	assertEqual(t, CountTotal("alpha"), uint64(3), "alpha not pruned to --max")
	assertEqual(t, CountTotal("beta"), uint64(2), "beta should not be pruned")
}

func inSlice(s string, list []string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}

	return false
}

func TestSegmentsPruneMaxAge(t *testing.T) {
	setup("")
	defer Close()
	enableSegments(t)

	config.UseMessageDates = true
	config.MaxAgeInHours = 24
	defer func() {
		config.UseMessageDates = false
		config.MaxAgeInHours = 0
	}()

	store := func(seg string, age time.Duration) {
		b := []byte("Date: " + time.Now().Add(-age).Format(time.RFC1123Z) + "\r\nFrom: sender@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nHello\r\n")
		if _, err := Store(seg, &b, nil); err != nil {
			t.Fatal(err)
		}
	}

	// --max-age is applied to every segment
	for _, seg := range []string{segment.Default, "alpha", "beta"} {
		store(seg, 48*time.Hour)
		store(seg, time.Hour)
	}

	pruneMessages()

	for _, seg := range []string{segment.Default, "alpha", "beta"} {
		assertEqual(t, CountTotal(seg), uint64(1), "old message not pruned in segment "+seg)
	}
}
