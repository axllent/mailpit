package tools

import (
	"reflect"
	"testing"
)

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
	tests["this_is-a&^%%(*)@ test"] = "this_is-a test"

	for search, expected := range tests {
		res := CleanTag(search)
		if res != expected {
			t.Log("CleanTags error:", res, "!=", expected)
			t.Fail()
		}
	}
}
