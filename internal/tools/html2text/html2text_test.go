package html2text

import "testing"

func TestPlain(t *testing.T) {
	tests := map[string]string{}
	tests["this is a  test"] = "this is a test"
	tests["thiS IS a Test"] = "thiS IS a Test"
	tests["thiS IS a Test :-)"] = "thiS IS a Test :-)"
	tests["<h1>This is a test.</h1> "] = "This is a test."
	tests["<p>Paragraph 1</p><p>Paragraph 2</p>"] = "Paragraph 1 Paragraph 2"
	tests["<h1>Heading</h1><p>Paragraph</p>"] = "Heading Paragraph"
	tests["<span>Alpha</span>bet <strong>chars</strong>"] = "Alphabet chars"
	tests["<span><b>A</b>lpha</span>bet  chars."] = "Alphabet chars."
	tests["<table><tr><td>First</td><td>Second</td></table>"] = "First Second"
	tests[`<h1>Heading</h1>
		<p>Paragraph</p>`] = "Heading Paragraph"
	tests[`<h1>Heading</h1><p>   <a href="https://github.com">linked text</a></p>`] = "Heading linked text"
	// broken html
	tests[`<h1>Heading</h3><p>   <a href="https://github.com">linked text.`] = "Heading linked text."

	for str, expected := range tests {
		res := Strip(str, false)
		if res != expected {
			t.Log("error:", res, "!=", expected)
			t.Fail()
		}
	}
}

func TestWithLinks(t *testing.T) {
	tests := map[string]string{}
	tests["this is a  test"] = "this is a test"
	tests["thiS IS a Test"] = "thiS IS a Test"
	tests["thiS IS a Test :-)"] = "thiS IS a Test :-)"
	tests["<h1>This is a test.</h1> "] = "This is a test."
	tests["<p>Paragraph 1</p><p>Paragraph 2</p>"] = "Paragraph 1 Paragraph 2"
	tests["<h1>Heading</h1><p>Paragraph</p>"] = "Heading Paragraph"
	tests["<span>Alpha</span>bet <strong>chars</strong>"] = "Alphabet chars"
	tests["<span><b>A</b>lpha</span>bet  chars."] = "Alphabet chars."
	tests["<table><tr><td>First</td><td>Second</td></table>"] = "First Second"
	tests["<h1>Heading</h1><p>Paragraph</p>"] = "Heading Paragraph"
	tests[`<h1>Heading</h1>
		<p>Paragraph</p>`] = "Heading Paragraph"
	tests[`<h1>Heading</h1><p>   <a href="https://github.com">linked text</a></p>`] = "Heading https://github.com linked text"
	// broken html
	tests[`<h1>Heading</h3><p>   <a href="https://github.com">linked text.`] = "Heading https://github.com linked text."

	for str, expected := range tests {
		res := Strip(str, true)
		if res != expected {
			t.Log("error:", res, "!=", expected)
			t.Fail()
		}
	}
}
