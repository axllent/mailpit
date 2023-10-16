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

func BenchmarkPlain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Strip(htmlTestData, false)
	}
}

func BenchmarkLinks(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Strip(htmlTestData, true)
	}
}

var htmlTestData = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" xmlns="http://www.w3.org/1999/xhtml" lang="en" xml:lang="en" style="font-family: sans-serif; -ms-text-size-adjust: 100%; -webkit-text-size-adjust: 100%; box-sizing: border-box;" xml:lang="en">
  <head>
    <meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
    <meta name="viewport" content="width=device-width" />
    <title>[axllent/mailpit] Run failed: .github/workflows/tests.yml - feature/swagger (284335a)</title>
    
  </head>
  <body style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot;; font-size: 14px; line-height: 1.5; color: #24292e; background-color: #fff; margin: 0;" bgcolor="#fff">
    <table align="center" class="container-sm width-full" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; max-width: 544px; margin-right: auto; margin-left: auto; width: 100% !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
        <td class="center p-3" align="center" valign="top" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 16px;">
          <center style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
            <table border="0" cellspacing="0" cellpadding="0" align="center" class="width-full container-md" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; max-width: 768px; margin-right: auto; margin-left: auto; width: 100% !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td align="center" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
              <table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="16" style="font-size: 16px; line-height: 16px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>

              <table border="0" cellspacing="0" cellpadding="0" align="left" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
                <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
                  <td class="text-left" style="box-sizing: border-box; text-align: left !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;" align="left">
                    <img src="https://github.githubassets.com/images/email/global/octocat-logo.png" alt="GitHub" width="32" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; border-style: none;" />
                    <h2 class="lh-condensed mt-2 text-normal" style="box-sizing: border-box; margin-top: 8px !important; margin-bottom: 0; font-size: 24px; font-weight: 400 !important; line-height: 1.25 !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
                        [axllent/mailpit] .github/workflows/tests.yml workflow run

                    </h2>
                  </td>
                </tr>
              </table>
              <table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="16" style="font-size: 16px; line-height: 16px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>

</td>
  </tr>
</table>
            <table width="100%" class="width-full" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; width: 100% !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
              <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
                <td class="border rounded-2 d-block" style="box-sizing: border-box; border-radius: 6px !important; display: block !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0; border: 1px solid #e1e4e8;">
                  <table align="center" class="width-full text-center" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; width: 100% !important; text-align: center !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
                    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
                      <td style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
                        <table border="0" cellspacing="0" cellpadding="0" align="center" class="width-full" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; width: 100% !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td align="center" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
                          
<table align="center" class="border-bottom width-full text-center" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; border-bottom-width: 1px !important; border-bottom-color: #e1e4e8 !important; border-bottom-style: solid !important; width: 100% !important; text-align: center !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td class="d-block px-3 pt-3 p-sm-4" style="box-sizing: border-box; display: block !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 24px;">
      <table border="0" cellspacing="0" cellpadding="0" align="center" class="width-full" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; width: 100% !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td align="center" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
        
    <img src="https://github.githubassets.com/images/email/icons/actions.png" width="56" height="56" alt="" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; border-style: none;" />
  <table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="12" style="font-size: 12px; line-height: 12px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>

<h3 class="lh-condensed" style="box-sizing: border-box; margin-top: 0; margin-bottom: 0; font-size: 20px; font-weight: 600; line-height: 1.25 !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">.github/workflows/tests.yml: No jobs were run</h3>
<table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="16" style="font-size: 16px; line-height: 16px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>



  <table border="0" cellspacing="0" cellpadding="0" align="center" class="width-full" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; width: 100% !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td align="center" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
    <table width="100%" border="0" cellspacing="0" cellpadding="0" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
      <table border="0" cellspacing="0" cellpadding="0" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
        <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
          <td align="center" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
              <!--[if mso]> <table><tr><td align="center" bgcolor="#28a745"> <![endif]-->
                <a href="https://github.com/axllent/mailpit/actions/runs/6522820865" target="_blank" rel="noopener noreferrer" class="btn btn-large btn-primary" style="background-color: #1f883d !important; box-sizing: border-box; color: #fff; text-decoration: none; position: relative; display: inline-block; font-size: inherit; font-weight: 500; line-height: 1.5; white-space: nowrap; vertical-align: middle; cursor: pointer; -webkit-user-select: none; user-select: none; border-radius: .5em; -webkit-appearance: none; appearance: none; box-shadow: 0 1px 0 rgba(27,31,35,.1),inset 0 1px 0 rgba(255,255,255,.03); transition: background-color .2s cubic-bezier(0.3, 0, 0.5, 1); font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: .75em 1.5em; border: 1px solid #1f883d;">View workflow run</a>
              <!--[if mso]> </td></tr></table> <![endif]-->
          </td>
        </tr>
      </table>
    </td>
  </tr>
</table>

</td>
  </tr>
</table>
  <table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="32" style="font-size: 32px; line-height: 32px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>


</td>
  </tr>
</table>
    </td>
  </tr>
</table>




</td>
  </tr>
</table>
                      </td>
                    </tr>
                  </table>
                </td>
              </tr>
            </table>
            <table border="0" cellspacing="0" cellpadding="0" align="center" class="width-full text-center" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; width: 100% !important; text-align: center !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td align="center" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
              <table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="16" style="font-size: 16px; line-height: 16px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>

              <table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="16" style="font-size: 16px; line-height: 16px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>

              <p class="f5 text-gray-light" style="box-sizing: border-box; margin-top: 0; margin-bottom: 10px; color: #6a737d !important; font-size: 14px !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">  </p><p style="font-size: small; -webkit-text-size-adjust: none; color: #666; box-sizing: border-box; margin-top: 0; margin-bottom: 10px; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">&#8212;<br style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;" />You are receiving this because you are subscribed to this thread.<br style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;" /><a href="https://github.com/settings/notifications" style="background-color: transparent; box-sizing: border-box; color: #0366d6; text-decoration: none; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">Manage your GitHub Actions notifications</a></p>

</td>
  </tr>
</table>
            <table border="0" cellspacing="0" cellpadding="0" align="center" class="width-full text-center" width="100%" style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; width: 100% !important; text-align: center !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <td align="center" style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">
  <table style="box-sizing: border-box; border-spacing: 0; border-collapse: collapse; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
  <tbody style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
    <tr style="box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">
      <td height="16" style="font-size: 16px; line-height: 16px; box-sizing: border-box; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important; padding: 0;">&#160;</td>
    </tr>
  </tbody>
</table>

  <p class="f6 text-gray-light" style="box-sizing: border-box; margin-top: 0; margin-bottom: 10px; color: #6a737d !important; font-size: 12px !important; font-family: -apple-system,BlinkMacSystemFont,&quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot; !important;">GitHub, Inc. &#12539;88 Colin P Kelly Jr Street &#12539;San Francisco, CA 94107</p>
</td>
  </tr>
</table>

          </center>
        </td>
      </tr>
    </table>
    <!-- prevent Gmail on iOS font size manipulation -->
   <div style="display: none; white-space: nowrap; box-sizing: border-box; font: 15px/0 apple-system, BlinkMacSystemFont, &quot;Segoe UI&quot;,Helvetica,Arial,sans-serif,&quot;Apple Color Emoji&quot;,&quot;Segoe UI Emoji&quot;;"> &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; &#160; </div>
  </body>
</html>`
