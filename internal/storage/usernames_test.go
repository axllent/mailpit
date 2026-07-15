package storage

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/jhillyerd/enmime/v2"
)

// storeWithUsername stores a simple message optionally authenticated as username.
func storeWithUsername(t *testing.T, i int, username *string) {
	t.Helper()

	msg := enmime.Builder().
		From(fmt.Sprintf("From %d", i), fmt.Sprintf("from-%d@example.com", i)).
		To("Inbox", "inbox@example.com").
		Subject(fmt.Sprintf("Message %d", i)).
		Text([]byte("This is the email body"))

	env, err := msg.Build()
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if err := env.Encode(buf); err != nil {
		t.Fatal(err)
	}

	b := buf.Bytes()
	if _, err := Store(&b, username); err != nil {
		t.Fatal(err)
	}
}

// TestUsernameSearch verifies the `username:` search filter that backs the
// per-username mailbox views.
func TestUsernameSearch(t *testing.T) {
	for _, tenantID := range []string{"", "MyServer 3", "host.example.com"} {
		tenantID = config.DBTenantID(tenantID)
		setup(tenantID)

		serviceA := "service-a"
		serviceB := "service-b"

		// 3 authenticated as service-a, 2 as service-b, 1 unauthenticated
		for i := range 3 {
			storeWithUsername(t, i, &serviceA)
		}
		for i := range 2 {
			storeWithUsername(t, i, &serviceB)
		}
		storeWithUsername(t, 0, nil)

		// username:service-a -> exactly the 3 service-a messages
		res, total, err := Search("username:service-a", "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, len(res), 3, "username:service-a result count")
		assertEqual(t, total, 3, "username:service-a total")
		for _, m := range res {
			assertEqual(t, m.Username, "service-a", "returned message username should match")
		}

		// username:service-b -> exactly the 2 service-b messages
		res, _, err = Search("username:service-b", "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, len(res), 2, "username:service-b result count")

		// exclusion: everything that is NOT service-a (2 service-b + 1 unauthenticated)
		res, _, err = Search("-username:service-a", "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, len(res), 3, "-username:service-a result count")
		for _, m := range res {
			if m.Username == "service-a" {
				t.Fatal("excluded username should not be present in -username:service-a results")
			}
		}

		// exact match only: a prefix must not match a longer username
		res, _, err = Search("username:service", "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, len(res), 0, "username: should be an exact match, not a prefix")

		// unknown username returns nothing
		res, _, err = Search("username:does-not-exist", "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, len(res), 0, "unknown username result count")

		// composes with other filters (mailbox + tag/subject scenario)
		res, _, err = Search("username:service-a subject:\"Message 0\"", "", 0, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		assertEqual(t, len(res), 1, "username composed with subject result count")

		Close()
	}
}

// TestGetAllUsernames verifies the distinct-username listing used to populate
// the mailbox switcher.
func TestGetAllUsernames(t *testing.T) {
	for _, tenantID := range []string{"", "host.example.com"} {
		tenantID = config.DBTenantID(tenantID)
		setup(tenantID)

		// empty to start
		assertEqual(t, len(GetAllUsernames()), 0, "no usernames expected on empty mailbox")

		serviceA := "service-a"
		serviceB := "service-b"

		storeWithUsername(t, 0, &serviceA)
		storeWithUsername(t, 1, &serviceA) // duplicate username
		storeWithUsername(t, 2, &serviceB)
		storeWithUsername(t, 3, nil) // unauthenticated -> excluded

		got := GetAllUsernames()
		assertEqual(t, len(got), 2, "distinct username count")
		// results are ordered alphabetically
		assertEqual(t, got[0], "service-a", "first username (sorted)")
		assertEqual(t, got[1], "service-b", "second username (sorted)")

		// also surfaced through StatsGet for the API/UI
		assertEqual(t, len(StatsGet().Usernames), 2, "StatsGet usernames count")

		Close()
	}
}
