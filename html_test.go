package html

import (
	"io/ioutil"
	"testing"
)

const SAMPLE_DOC = `
<!DOCTYPE html>
<html>
	<head>
		<title>Test Document</title>
		<meta name="referrer" content="origin">
		<link rel="stylesheet" type="text/css" href="foo.css?123456">
		<script type="text/javascript">
			function hide(id) {
				var el = document.getElementById(id);
				if (el) { el.style.visibility = 'hidden'; }
			}
		</script>
	</head>
	<body>
		<div class="container">
			<h1 id="my-header">Hello World!</h1>
			<a href="/internal" class="my-link-class highlight">Test Internal Link</a>
			<a href="http://test/external" class="highlight" target="_blank">Test External Link</a>
		</div>
		<div class="footer">
			<!-- XML COMMENTS BITCHES. -->
		</div>
	</body>
</html>`

const SNIPPET = `<div id="first"><br><p><h1>Test!</h1></p></div><div id="second"><h2>Test 2!</h2></div>`
const SNIPPET_INVALID = `<div id="first"><br><p><h1>Test!</p></h1></div><div id="second"><h2>Test 2!</h2></div>`

func TestParsingSnippet(t *testing.T) {
	doc, parseError := Parse(SNIPPET)
	if parseError != nil {
		t.Error(parseError.Error())
		t.FailNow()
	}

	if len(doc.Children) == 0 {
		t.Error("doc children length is 0")
		t.FailNow()
	}

	if doc.Children[0].Attributes["id"] != "first" {
		t.Errorf("Invalid first child: %s", doc.Children[0].ToString())
		t.FailNow()
	}

	if doc.Children[1].Attributes["id"] != "second" {
		t.Errorf("Invalid first child: %s", doc.Children[1].ToString())
		t.FailNow()
	}
}

func TestParsingDocument(t *testing.T) {
	doc, parse_eror := Parse(SAMPLE_DOC)
	if parse_eror != nil {
		t.Error(parse_eror.Error())
		t.FailNow()
	}

	if len(doc.NonTextChildren()) == 0 {
		t.Error("doc children length is 0")
		t.FailNow()
	}

	text_elements := doc.GetElementsByTagName("text")
	if len(text_elements) == 0 {
		t.Error("`text` element count is 0")
		t.FailNow()
	}

	div_elements := doc.GetElementsByTagName(ELEMENT_DIV)
	if len(div_elements) != 2 {
		t.Errorf(`GetElementsByTagName("div") count is %d, expected 2`, len(div_elements))
		t.FailNow()
	}

	a_elements := doc.GetElementsByTagName(ELEMENT_A)
	if len(a_elements) != 2 {
		t.Errorf(`GetElementsByTagName("a") count is %d, expected 2`, len(a_elements))
		t.FailNow()
	}

	class_elements := doc.GetElementsByClassName("highlight")
	if len(class_elements) != 2 {
		t.Errorf(`GetElementsByClassName("highlight") count is %d, expected 2`, len(class_elements))
		t.FailNow()
	}

	predicate_elements := doc.GetElementsByPredicate(func(e *Element) bool {
		return e.Parent.ElementName == ELEMENT_DIV && e.Parent.HasClass("container")
	})

	if len(predicate_elements) == 0 {
		t.Errorf("GetElementsByPredicate returned zero results")
		t.FailNow()
	}

	sample_element := predicate_elements[0]
	sample_element_path := sample_element.GetPath()
	if len(sample_element_path) == 0 {
		t.Error("GetPath() produced a 0 length path.")
		t.FailNow()
	}

	header := doc.GetElementById("my-header")
	if header == nil {
		t.Error("GetElementById() returned a nil header")
		t.FailNow()
	}
	if header.GetId() != "my-header" {
		t.Error("GetElementById() returned an element with the wrong Id")
		t.FailNow()
	}
}

func TestParsingMocks(t *testing.T) {
	mock_files := []string{
		"news.ycombinator.com.html",
		"nytimes.com.html",
		"blendlabs.clean.html",
		"blendlabs.com.html",
	}

	for _, mock_file := range mock_files {
		corpus := readFileContents("mocks/" + mock_file)
		_, parseError := Parse(corpus)

		if parseError != nil {
			t.Errorf("error with %s: %s", mock_file, parseError.Error())
			t.FailNow()
		}
	}
}

func TestParsingInvalid(t *testing.T) {
	_, parseError := ParseStrict(SNIPPET_INVALID)
	if parseError == nil {
		t.Error("Should have errored.")
		t.FailNow()
	}
}

func TestElementStack(t *testing.T) {
	stack := &elementStack{}

	if stack.Count != 0 {
		t.Errorf("initial stack storage count is not 0, is: %d", stack.Count)
		t.FailNow()
	}

	stack.Push(Element{ElementName: "br", IsVoid: true})

	if stack.Count != 1 {
		t.Errorf("stack storage count is not 1, is: %d", stack.Count)
		t.FailNow()
	}

	stack.Push(Element{ElementName: "div", Attributes: map[string]string{"class": "first"}})

	if stack.Count != 2 {
		t.Errorf("stack storage count is not 2, is: %d", stack.Count)
		t.FailNow()
	}

	stack.Push(Element{ElementName: "div", Attributes: map[string]string{"class": "second"}})

	if stack.Count != 3 {
		t.Errorf("stack storage count is not 3, is: %d", stack.Count)
		t.FailNow()
	}

	stack_string := stack.ToString()
	if stack_string != "br > div > div" {
		t.Errorf("stack .ToString() invalid: %s", stack_string)
		t.FailNow()
	}

	duplicated := stack.Duplicate()
	duplicated_stack_string := duplicated.ToString()
	if stack_string != duplicated_stack_string {
		t.Errorf("Duplicate() error: %s != %s", stack_string, duplicated_stack_string)
		t.FailNow()
	}

	if stack.Peek().ElementName != "div" {
		t.Error("top of stack is not a `div`")
		t.FailNow()
	}

	if stack.Count != 3 {
		t.Errorf("stack storage count is not 3, is: %d", stack.Count)
		t.FailNow()
	}

	div := stack.Pop()
	if div.ElementName != "div" {
		t.Error("first popped element should be a div")
		t.FailNow()
	}

	if div.Attributes["class"] != "second" {
		t.Error("first popped element `class` should be `second`")
		t.FailNow()
	}

	if stack.Count != 2 {
		t.Error("stack count should be 2 after popping first element.")
		t.FailNow()
	}
}

func TestReadUntilTag(t *testing.T) {
	cursor := 0
	valid := "      this is a test of reading until the tag <area/>"

	results, results_err := readUntilTag([]rune(valid), &cursor)
	if results_err != nil {
		t.Error(results_err.Error())
		t.FailNow()
	}

	if string(results) != "      this is a test of reading until the tag " {
		t.Error("Incorrect results: '" + string(results) + "'")
		t.FailNow()
	}

	cursor = 0
	no_tag := "there is no tag."

	results, results_err = readUntilTag([]rune(no_tag), &cursor)
	if results_err != nil {
		t.Error(results_err.Error())
		t.FailNow()
	}
	if string(results) != no_tag {
		t.Error("Incorrect results.")
		t.FailNow()
	}

	cursor = 0
	only_tag := "<a href='things.html'>things</a>"
	results, results_err = readUntilTag([]rune(only_tag), &cursor)
	if results_err != nil {
		t.Error(results_err.Error())
		t.FailNow()
	}
	if string(results) != EMPTY {
		t.Error("Incorrect results.")
		t.FailNow()
	}

	cursor = 0
	starts_tag := "<br/> more text ..."
	results, results_err = readUntilTag([]rune(starts_tag), &cursor)
	if results_err != nil {
		t.Error(results_err.Error())
		t.FailNow()
	}
	if string(results) != EMPTY {
		t.Error("Incorrect results.")
		t.FailNow()
	}
}

func TestReadUntilScriptTagClose(t *testing.T) {
	test_cases := map[string]string{
		`var a = "abc";</script>`:      `var a = "abc";`,
		`alert('</script>');</script>`: `alert('</script>');`,

		`//</script>
		var foo = "bar";
		</script>`: `//</script>
		var foo = "bar";
		`,

		`var foo = 'bar';
		/* this is a block 
		comment and is annoying */
		foo = 'baz';
		</script>`: `var foo = 'bar';
		/* this is a block 
		comment and is annoying */
		foo = 'baz';
		`,
		`
			function hide(id) {
				var el = document.getElementById(id);
				if (el) { el.style.visibility = 'hidden'; }
			}
		</script>`: `
			function hide(id) {
				var el = document.getElementById(id);
				if (el) { el.style.visibility = 'hidden'; }
			}
		`,
	}

	for test, expected := range test_cases {
		cursor := 0
		results, results_err := readUntilScriptTagClose([]rune(test), &cursor, "text/javascript")
		if results_err != nil {
			t.Error("error occurred.")
			t.FailNow()

		}
		if len(results) == 0 {
			t.Error("empty results.")
			t.FailNow()
		}

		if expected != string(results) {
			t.Errorf("expected: '%s' actual: '%s'", expected, string(results))
			t.FailNow()
		}
	}
}

func TestReadWhitespace(t *testing.T) {
	test_string := "     \n\t     this is a test string ..."
	cursor := 0
	results, results_err := readWhitespace([]rune(test_string), &cursor)
	if results_err != nil {
		t.Error(results_err.Error())
		t.FailNow()
	}

	if string(results) != "     \n\t     " {
		t.Error("Incorrect results.")
		t.FailNow()
	}
}

func TestElementEqualTo(t *testing.T) {
	reference := Element{ElementName: ELEMENT_A, IsVoid: false, IsComment: false, Attributes: map[string]string{"href": "home.html", "class": "content"}}

	correct := Element{ElementName: ELEMENT_A, IsVoid: false, IsComment: false, Attributes: map[string]string{"href": "home.html", "class": "content"}}

	href_wrong := Element{ElementName: ELEMENT_A, IsVoid: false, IsComment: false, Attributes: map[string]string{"href": "home2.html", "class": "content"}}
	element_name_wrong := Element{ElementName: ELEMENT_DIV, IsVoid: false, IsComment: false, Attributes: map[string]string{"href": "home.html", "class": "content"}}
	class_wrong := Element{ElementName: ELEMENT_DIV, IsVoid: false, IsComment: false, Attributes: map[string]string{"href": "home.html", "class": "not-content"}}

	if !reference.EqualTo(correct) {
		t.Error("EqualTo failed for correct case.")
		t.FailNow()
	}

	if reference.EqualTo(href_wrong) {
		t.Error("EqualTo failed for attribute 'href' case.")
		t.FailNow()
	}

	if reference.EqualTo(element_name_wrong) {
		t.Error("EqualTo failed for attribute `ElementName` case.")
		t.FailNow()
	}

	if reference.EqualTo(class_wrong) {
		t.Error("EqualTo failed for attribute attribute `class` case.")
		t.FailNow()
	}
}

func TestReadTag(t *testing.T) {
	testCases := map[string]Element{
		"<!DOCTYPE>":                 Element{ElementName: ELEMENT_DOCTYPE, IsVoid: true, Attributes: map[string]string{}},
		"<!DOCTYPE html>":            Element{ElementName: ELEMENT_DOCTYPE, IsVoid: true, Attributes: map[string]string{"html": ""}},
		"<!-- this is a comment -->": Element{ElementName: ELEMENT_INTERNAL_XML_COMMENT, IsVoid: true, IsComment: true, InnerHTML: " this is a comment ", Attributes: map[string]string{}},
		"<br>":                                                                    Element{ElementName: ELEMENT_BR, IsVoid: true, Attributes: map[string]string{}},
		"<br/>":                                                                   Element{ElementName: ELEMENT_BR, IsVoid: true, Attributes: map[string]string{}},
		"</div>":                                                                  Element{ElementName: ELEMENT_DIV, IsVoid: false, IsClose: true, Attributes: map[string]string{}},
		"</ div>":                                                                 Element{ElementName: ELEMENT_DIV, IsVoid: false, IsClose: true, Attributes: map[string]string{}},
		"< /div>":                                                                 Element{ElementName: ELEMENT_DIV, IsVoid: false, IsClose: true, Attributes: map[string]string{}},
		"<div class=\"\">":                                                        Element{ElementName: ELEMENT_DIV, Attributes: map[string]string{"class": ""}},
		"<div class=\"content\">":                                                 Element{ElementName: ELEMENT_DIV, Attributes: map[string]string{"class": "content"}},
		"<div class=\"with='quotes'\">":                                           Element{ElementName: ELEMENT_DIV, Attributes: map[string]string{"class": "with='quotes'"}},
		"<div class='with=\"escaped_quotes\"'>":                                   Element{ElementName: ELEMENT_DIV, Attributes: map[string]string{"class": "with=\"escaped_quotes\""}},
		"<a class=\"my-link\" href=\"/test/route\" />":                            Element{ElementName: ELEMENT_A, IsVoid: true, Attributes: map[string]string{"class": "my-link", "href": "/test/route"}},
		"<section class=\"module streamline-automate type-standard\" style=\"\">": Element{ElementName: ELEMENT_SECTION, Attributes: map[string]string{"class": "module streamline-automate type-standard", "style": ""}},
		`<a href="http://test/external" class="highlight" target="_blank">`:       Element{ElementName: ELEMENT_A, Attributes: map[string]string{"href": "http://test/external", "class": "highlight", "target": "_blank"}},
	}

	for tag, expectedResult := range testCases {
		cursor := 0
		actualResult, parseError := readTag([]rune(tag), &cursor)

		if parseError != nil {
			t.Error(parseError.Error())
			t.FailNow()
		}
		if !expectedResult.EqualTo(*actualResult) {
			t.Error("Invalid parsed tag results.")
			t.Errorf("\tExpected : %s", expectedResult.ToString())
			t.Errorf("\tActual   : %s", actualResult.ToString())
			t.Fail()
		}
	}
}

func TestGetInnerText(t *testing.T) {
	document, _ := Parse(SNIPPET)
	text := document.GetInnerText()
	if len(text) == 0 {
		t.Error("GetInnerText() produced 0 length text")
		t.FailNow()
	}

	if text != "Test!Test 2!" {
		t.Errorf("GetInnerText() produced the wrong text: %s", text)
		t.FailNow()
	}
}

func readFileContents(filename string) string {
	reader, _ := ioutil.ReadFile(filename)
	return string(reader)
}
