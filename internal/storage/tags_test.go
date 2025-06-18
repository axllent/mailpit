package storage

import (
	"fmt"
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
)

func TestTags(t *testing.T) {

	for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
		tenantID = config.DBTenantID(tenantID)

		setup(tenantID)

		if tenantID == "" {
			t.Log("Testing tags")
		} else {
			t.Logf("Testing tags (tenant %s)", tenantID)
		}

		ids := []string{}

		for i := 0; i < 10; i++ {
			id, err := Store(&testMimeEmail, nil)
			if err != nil {
				t.Log("error ", err)
				t.Fail()
			}
			ids = append(ids, id)
		}

		for i := 0; i < 10; i++ {
			if _, err := SetMessageTags(ids[i], []string{fmt.Sprintf("Tag-%d", i)}); err != nil {
				t.Log("error ", err)
				t.Fail()
			}
		}

		for i := 0; i < 10; i++ {
			message, err := GetMessage(ids[i])
			if err != nil {
				t.Log("error ", err)
				t.Fail()
			}

			if len(message.Tags) != 1 || message.Tags[0] != fmt.Sprintf("Tag-%d", i) {
				t.Fatal("Message tags do not match")
			}
		}

		if err := DeleteAllMessages(); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		// test 20 tags
		id, err := Store(&testMimeEmail, nil)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		newTags := []string{}
		for i := 0; i < 20; i++ {
			// pad number with 0 to ensure they are returned alphabetically
			newTags = append(newTags, fmt.Sprintf("AnotherTag %02d", i))
		}
		if _, err := SetMessageTags(id, newTags); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		returnedTags := getMessageTags(id)
		assertEqual(t, strings.Join(newTags, "|"), strings.Join(returnedTags, "|"), "Message tags do not match")

		// remove first tag
		if err := deleteMessageTag(id, newTags[0]); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		returnedTags = getMessageTags(id)
		assertEqual(t, strings.Join(newTags[1:], "|"), strings.Join(returnedTags, "|"), "Message tags do not match after deleting 1")

		// remove all tags
		if err := DeleteAllMessageTags(id); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		returnedTags = getMessageTags(id)
		assertEqual(t, "", strings.Join(returnedTags, "|"), "Message tags should be empty")

		// apply the same tag twice
		if _, err := SetMessageTags(id, []string{"Duplicate Tag", "Duplicate Tag"}); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		returnedTags = getMessageTags(id)
		assertEqual(t, "Duplicate Tag", strings.Join(returnedTags, "|"), "Message tags should be duplicated")
		if err := DeleteAllMessageTags(id); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		// apply tag with invalid characters
		if _, err := SetMessageTags(id, []string{"Dirty! \"Tag\""}); err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		returnedTags = getMessageTags(id)
		assertEqual(t, "Dirty Tag", strings.Join(returnedTags, "|"), "Dirty message tag did not clean as expected")
		if err := DeleteAllMessageTags(id); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		// Check deleted message tags also prune the tags database
		allTags := GetAllTags()
		assertEqual(t, "", strings.Join(allTags, "|"), "Tags did not delete as expected")

		if err := DeleteAllMessages(); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		// test 20 tags
		id, err = Store(&testTagEmail, nil)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		returnedTags = getMessageTags(id)
		assertEqual(t, "BccTag|CcTag|FromFag|ToTag|X-tag1|X-tag2", strings.Join(returnedTags, "|"), "Tags not detected correctly")
		if err := DeleteAllMessageTags(id); err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		Close()
	}

}
func TestUsernameAutoTagging(t *testing.T) {
	setup("")
	defer Close()

	username := "testuser"

	t.Run("Auto-tagging enabled", func(t *testing.T) {
		config.TagsUsername = true
		id, err := Store(&testTextEmail, &username)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
		msg, err := GetMessage(id)
		if err != nil {
			t.Fatalf("GetMessage failed: %v", err)
		}
		found := false
		for _, tag := range msg.Tags {
			if tag == username {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected username '%s' in tags, got %v", username, msg.Tags)
		}
	})

	t.Run("Auto-tagging disabled", func(t *testing.T) {
		config.TagsUsername = false
		id, err := Store(&testTextEmail, &username)
		if err != nil {
			t.Fatalf("Store failed: %v", err)
		}
		msg, err := GetMessage(id)
		if err != nil {
			t.Fatalf("GetMessage failed: %v", err)
		}
		for _, tag := range msg.Tags {
			if tag == username {
				t.Errorf("Did not expect username '%s' in tags when disabled, got %v", username, msg.Tags)
			}
		}
	})
}
