package storage

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

func xTagsMessage(to string, tags []string) []byte {
	return []byte("From: a@example.com\r\nTo: " + to + "\r\nSubject: tags\r\nX-Tags: " + strings.Join(tags, ", ") + "\r\n\r\nbody\r\n")
}

func numberedTags(prefix string, n int) []string {
	tags := make([]string, n)
	for i := range tags {
		tags[i] = fmt.Sprintf("%s-%05d", prefix, i)
	}

	return tags
}

func TestUniqueTags(t *testing.T) {
	got := uniqueTags([]string{"Zebra", "apple", "ZEBRA", " Apple ", "!!!", "", "mango"})
	want := []string{"Zebra", "apple", "mango"}

	// the first of each tag is kept, in the order given (unlike sortedUniqueTags)
	if !slices.Equal(got, want) {
		t.Fatalf("uniqueTags = %v, want %v", got, want)
	}

	if got := sortedUniqueTags([]string{"Zebra", "apple", "ZEBRA"}); !slices.Equal(got, []string{"Zebra", "apple"}) {
		t.Fatalf("sortedUniqueTags = %v", got)
	}
}

// A message can have at most MaxTagsPerMessage tags, however many its headers ask for.
func TestMessageTagLimit(t *testing.T) {
	setup("")

	tests := []struct {
		name string
		tags int
		want int
	}{
		{"control: a few tags", 3, 3},
		{"control: exactly the limit", MaxTagsPerMessage, MaxTagsPerMessage},
		{"exploit: one over the limit", MaxTagsPerMessage + 1, MaxTagsPerMessage},
		{"exploit: thousands of tags", 5000, MaxTagsPerMessage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := xTagsMessage("x@example.com", numberedTags("tag", tt.tags))

			id, err := Store(&raw, nil)
			if err != nil {
				t.Fatal(err)
			}

			stored := getMessageTags(id)
			if len(stored) != tt.want {
				t.Fatalf("%d tags requested, %d stored, want %d", tt.tags, len(stored), tt.want)
			}
			if !slices.IsSorted(stored) {
				t.Fatal("the stored tags are not sorted")
			}
		})
	}
}

// When there are too many tags, the ones from the most trusted source are kept. Here the username
// and the plus-address tag sort after every X-Tags entry, so a limit applied after sorting would
// drop them.
func TestMessageTagLimitKeepsHigherPriorityTags(t *testing.T) {
	setup("")

	orig := config.TagsUsername
	config.TagsUsername = true
	t.Cleanup(func() { config.TagsUsername = orig })

	username := "zz-user"
	raw := xTagsMessage("dev+zz-plus@example.com", numberedTags("aa", 300))

	id, err := Store(&raw, &username)
	if err != nil {
		t.Fatal(err)
	}

	stored := getMessageTags(id)
	if len(stored) != MaxTagsPerMessage {
		t.Fatalf("%d tags stored, want %d", len(stored), MaxTagsPerMessage)
	}
	for _, want := range []string{"zz-user", "zz-plus"} {
		if !slices.Contains(stored, want) {
			t.Errorf("the tag %q was displaced by X-Tags entries, stored: %v", want, stored)
		}
	}

	// control: with only a few X-Tags, nothing is lost
	raw = xTagsMessage("dev+zz-plus@example.com", numberedTags("aa", 3))
	id, err = Store(&raw, &username)
	if err != nil {
		t.Fatal(err)
	}
	if got := getMessageTags(id); len(got) != 5 {
		t.Fatalf("expected 5 tags (3 X-Tags, the plus-address and the username), got %v", got)
	}
}

// The tags the API sets replace all of a message's tags, so asking for too many is an error
// rather than a silent truncation.
func TestSetMessageTagsLimit(t *testing.T) {
	setup("")

	raw := xTagsMessage("x@example.com", nil)
	id, err := Store(&raw, nil)
	if err != nil {
		t.Fatal(err)
	}

	// control
	if _, err := SetMessageTags(id, numberedTags("api", MaxTagsPerMessage)); err != nil {
		t.Fatalf("setting %d tags: %v", MaxTagsPerMessage, err)
	}
	if got := len(getMessageTags(id)); got != MaxTagsPerMessage {
		t.Fatalf("%d tags set", got)
	}

	// exploit
	if _, err := SetMessageTags(id, numberedTags("api", MaxTagsPerMessage+1)); err == nil {
		t.Fatalf("setting %d tags did not fail", MaxTagsPerMessage+1)
	}
	// and the message is left as it was
	if got := len(getMessageTags(id)); got != MaxTagsPerMessage {
		t.Fatalf("a failed request changed the tags, %d now set", got)
	}

	// duplicates and invalid tags are not counted
	many := append(numberedTags("api", MaxTagsPerMessage), numberedTags("api", MaxTagsPerMessage)...)
	many = append(many, "!!!", "", strings.ToUpper(many[0]))
	if _, err := SetMessageTags(id, many); err != nil {
		t.Fatalf("duplicates and invalid tags counted towards the limit: %v", err)
	}
}

// Reading and storing a huge X-Tags header must not cost more than the tags that can be kept.
// 60,000 tags took almost 10 seconds to store before the limit.
func TestHugeXTagsHeaderIsCheap(t *testing.T) {
	setup("")

	raw := xTagsMessage("x@example.com", numberedTags("tag", 60000))

	start := time.Now()
	id, err := Store(&raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("storing a message with 60,000 X-Tags took %v", d)
	}
	if got := len(getMessageTags(id)); got != MaxTagsPerMessage {
		t.Fatalf("%d tags stored", got)
	}
}
