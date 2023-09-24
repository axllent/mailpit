package storage

import (
	"fmt"
	"testing"
)

func TestTags(t *testing.T) {
	setup()
	defer Close()

	t.Log("Testing setting & getting tags")

	ids := []string{}

	for i := 0; i < 10; i++ {
		id, err := Store(testMimeEmail)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}
		ids = append(ids, id)
	}

	for i := 0; i < 10; i++ {
		if err := SetTags(ids[i], []string{fmt.Sprintf("Tag-%d", i)}); err != nil {
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
}
