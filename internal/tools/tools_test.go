package tools

import (
	"net"
	"reflect"
	"testing"
)

func TestIsInternalIP(t *testing.T) {
	internal := []string{
		"127.0.0.1",       // loopback
		"::1",             // IPv6 loopback
		"10.0.0.1",        // private
		"172.16.0.1",      // private
		"192.168.1.1",     // private
		"169.254.1.1",     // link-local unicast
		"fe80::1",         // IPv6 link-local
		"0.0.0.0",         // unspecified
		"224.0.0.1",       // multicast
		"100.64.0.1",      // CGNAT start
		"100.127.255.255", // CGNAT end
	}
	external := []string{
		"8.8.8.8",
		"1.1.1.1",
		"100.128.0.1", // just outside CGNAT range
	}

	for _, s := range internal {
		ip := net.ParseIP(s)
		if !IsInternalIP(ip) {
			t.Errorf("expected %s to be internal", s)
		}
	}
	for _, s := range external {
		ip := net.ParseIP(s)
		if IsInternalIP(ip) {
			t.Errorf("expected %s to be external", s)
		}
	}
}

func TestIsValidLinkURL(t *testing.T) {
	valid := []string{
		"http://example.com",
		"https://example.com",
		"https://example.com/path?q=1#anchor",
	}
	invalid := []string{
		"",
		"ftp://example.com",
		"example.com",
		"//example.com",
		"https://",
	}

	for _, s := range valid {
		if !IsValidLinkURL(s) {
			t.Errorf("expected %q to be a valid link URL", s)
		}
	}
	for _, s := range invalid {
		if IsValidLinkURL(s) {
			t.Errorf("expected %q to be an invalid link URL", s)
		}
	}
}

func TestArgsParser(t *testing.T) {
	tests := map[string][]string{}
	tests["this is a test"] = []string{"this", "is", "a", "test"}
	tests["\"this is\" a test"] = []string{"this is", "a", "test"}
	tests["!\"this is\" a test"] = []string{"!this is", "a", "test"}
	tests["subject:this is a test"] = []string{"subject:this", "is", "a", "test"}
	tests["subject:\"this is\" a test"] = []string{"subject:this is", "a", "test"}
	tests["subject:\"this is\" \"a test\""] = []string{"subject:this is", "a test"}
	tests["subject:\"this 'is\" \"a test\""] = []string{"subject:this 'is", "a test"}
	tests["subject:\"this 'is a test"] = []string{"subject:this 'is a test"}
	tests["\"this is a test\"=\"this is a test\""] = []string{"this is a test=this is a test"}

	for search, expected := range tests {
		res := ArgsParser(search)
		if !reflect.DeepEqual(res, expected) {
			t.Log("Args parser error:", res, "!=", expected)
			t.Fail()
		}
	}
}

func TestCleanTag(t *testing.T) {
	tests := map[string]string{}
	tests["this is a test"] = "this is a test"
	tests["thiS IS a Test"] = "thiS IS a Test"
	tests["thiS IS a Test :-)"] = "thiS IS a Test -"
	tests["  thiS 99     IS a Test :-)"] = "thiS 99 IS a Test -"
	tests["this_is-a test "] = "this_is-a test"
	tests["this_is-a&^%%(*)@ test"] = "this_is-a @ test"
	tests["this is a long tag title with more than 100 characters, which should get automatically truncated to 100 characters"] = "this is a long tag title with more than 100 characters which should get automatically truncated to 1"

	for search, expected := range tests {
		res := CleanTag(search)
		if res != expected {
			t.Log("CleanTags error:", res, "!=", expected)
			t.Fail()
		}
	}
}

func TestSnippets(t *testing.T) {
	tests := map[string]string{}
	tests["this is a  test"] = "this is a test"
	tests["thiS IS a Test"] = "thiS IS a Test"
	tests["thiS IS a Test :-)"] = "thiS IS a Test :-)"
	tests["<h1>This is a test.</h1> "] = "This is a test."
	tests["this_is-a     test "] = "this_is-a test"
	tests["this_is-a&^%%(*)@ test"] = "this_is-a&^%%(*)@ test"
	tests["<h1>Heading</h1><p>Paragraph</p>"] = "Heading Paragraph"
	tests[`<h1>Heading</h1>
		<p>Paragraph</p>`] = "Heading Paragraph"
	tests[`<h1>Heading</h1><p>   <a href="https://github.com">linked text</a></p>`] = "Heading linked text"
	// broken html
	tests[`<h1>Heading</h3><p>   <a href="https://github.com">linked text.`] = "Heading linked text."
	// truncation to 200 chars + ...
	tests["abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789"] = "abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmnopqrstuvwxyx0123456789 abcdefghijklmno..."

	for str, expected := range tests {
		res := CreateSnippet(str, str)
		if res != expected {
			t.Log("CreateSnippet error:", res, "!=", expected)
			t.Fail()
		}
	}
}

func TestListUnsubscribeParser(t *testing.T) {
	tests := map[string]bool{}

	// should pass
	tests["<mailto:unsubscribe@example.com>"] = true
	tests["<https://example.com>"] = true
	tests["<HTTPS://EXAMPLE.COM>"] = true
	tests["<mailto:unsubscribe@example.com>, <http://example.com>"] = true
	tests["<mailto:unsubscribe@example.com>, <https://example.com>"] = true
	tests["<https://example.com>, <mailto:unsubscribe@example.com>"] = true
	tests["<https://example.com> , 		<mailto:unsubscribe@example.com>"] = true
	tests["<https://example.com> ,<mailto:unsubscribe@example.com>"] = true
	tests["<mailto:unsubscribe@example.com>,<https://example.com>"] = true
	tests[`<https://example.com> ,
		 <mailto:unsubscribe@example.com>`] = true
	tests["<mailto:unsubscribe@example.com?subject=unsubscribe%20me>"] = true
	tests["(Use this command to get off the list) <mailto:unsubscribe@example.com?subject=unsubscribe%20me>"] = true
	tests["<mailto:unsubscribe@example.com> (Use this command to get off the list)"] = true
	tests["(Use this command to get off the list) <mailto:unsubscribe@example.com>, (Click this link to unsubscribe) <http://example.com>"] = true

	// should fail
	tests["mailto:unsubscribe@example.com"] = false                                                // no <>
	tests["<mailto::unsubscribe@example.com>"] = false                                             // ::
	tests["https://example.com/"] = false                                                          // no <>
	tests["mailto:unsubscribe@example.com, <https://example.com/>"] = false                        // no <>
	tests["<MAILTO:unsubscribe@example.com>"] = false                                              // capitals
	tests["<mailto:unsubscribe@example.com>, <mailto:test2@example.com>"] = false                  // two emails
	tests["<http://exampl\\e2.com>, <http://example2.com>"] = false                                // two links
	tests["<http://example.com>, <mailto:unsubscribe@example.com>, <http://example2.com>"] = false // two links
	tests["<mailto:unsubscribe@example.com>, <example.com>"] = false                               // no mailto || http(s)
	tests["<mailto: unsubscribe@example.com>, <unsubscribe@lol.com>"] = false                      // space
	tests["<mailto:unsubscribe@example.com?subject=unsubscribe me>"] = false                       // space
	tests["<http:///example.com>"] = false                                                         // http:///

	for search, expected := range tests {
		_, err := ListUnsubscribeParser(search)
		hasError := err != nil
		if expected == hasError {
			if err != nil {
				t.Logf("ListUnsubscribeParser: %v", err)
			} else {
				t.Logf("ListUnsubscribeParser: \"%s\" expected: %v", search, expected)
			}
			t.Fail()
		}
	}
}
