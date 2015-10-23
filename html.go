package html

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

//--------------------------------------------------------------------------------
// MAIN EXPORTED METHODS
//--------------------------------------------------------------------------------

func Parse(body string) (Element, error) {
	parentElement := Element{IsRoot: true}
	tagStack := &elementStack{}
	cursor := 0
	childrenError := parseChildren(&parentElement, []rune(body), &cursor, tagStack)
	return parentElement, childrenError
}

func parseChildren(parentElement *Element, body []rune, cursor *int, tagStack *elementStack) error {
	if len(body) == 0 {
		return nil
	}

	parse_start := *cursor
	for *cursor < len(body) {
		results, results_err := readUntilTag(body, cursor)
		if results_err != nil {
			return results_err
		}

		if len(results) > 0 && !isContinuousWhitespace(results) {
			new_text_node := newTextNode(results)
			parentElement.AddChild(new_text_node)
		}

		read_tag, read_tag_error := readTag(body, cursor)
		if read_tag_error != nil {
			return read_tag_error
		}

		if read_tag.IsClose {
			expected_tag := tagStack.Peek()
			if expected_tag.ElementName == read_tag.ElementName {
				tagStack.Pop()
				parentElement.InnerHTML = string(body[parse_start:*cursor])
				return nil
			} else {
				error_text := fmt.Sprintf("unexpected close </%s> (expected </%s>) on line: %d", read_tag.ElementName, expected_tag.ElementName, countNewlinesBefore(string(body), *cursor))
				error_text = error_text + fmt.Sprintf("\ncurrent path: %s", tagStack.ToString())
				return errors.New(error_text)
			}
		} else if read_tag.IsVoid {
			parentElement.AddChild(read_tag)
		} else if read_tag.ElementName == "script" { //script tags are a black hole of misery and pain.
			script_type := "text/javascript"
			tag_script_type, has_script_type := read_tag.Attributes["type"]

			if has_script_type {
				script_type = tag_script_type
			}

			script_contents, script_error := readUntilScriptTagClose(body, cursor, script_type)
			if script_error != nil {
				return script_error
			}

			script_body := newTextNode(script_contents)
			read_tag.AddChild(script_body)
			parentElement.AddChild(read_tag)
		} else {
			new_stack := tagStack.Duplicate()
			new_stack.Push(*read_tag)
			parse_children_error := parseChildren(read_tag, body, cursor, new_stack)
			parentElement.AddChild(read_tag)
			if parse_children_error != nil {
				return parse_children_error
			}
		}
	}
	parentElement.InnerHTML = string(body[parse_start:*cursor])
	return nil
}

func countNewlinesBefore(body string, cursorPosition int) int {
	count := 0
	for x := 0; x < cursorPosition; x++ {
		c := body[x]
		if c == '\n' {
			count++
		}
	}

	return count
}

func newTextNode(text []rune) *Element {
	return &Element{ElementName: "text", IsText: true, IsVoid: true, InnerHTML: string(text)}
}

//--------------------------------------------------------------------------------
// META
//--------------------------------------------------------------------------------

const (
	EMPTY = ""
)

//known void elements
var (
	KNOWN_VOID_ELEMENTS = map[string]bool{
		"area":     true,
		"base":     true,
		"br":       true,
		"col":      true,
		"embed":    true,
		"hr":       true,
		"img":      true,
		"input":    true,
		"keygen":   true,
		"link":     true,
		"menuitem": true,
		"meta":     true,
		"param":    true,
		"source":   true,
		"track":    true,
		"wbr":      true,
	}
)

//--------------------------------------------------------------------------------
// TYPES: ELEMENT
//--------------------------------------------------------------------------------

type Element struct {
	ElementName string
	Parent      *Element
	InnerHTML   string
	Attributes  map[string]string
	Children    []Element
	IsText      bool
	IsVoid      bool
	IsComment   bool
	IsRoot      bool
	IsClose     bool
	IsData      bool
}

func (e *Element) AddChild(newChild *Element) {
	newChild.Parent = e
	e.Children = append(e.Children, *newChild)
}

func (e Element) Flatten() []Element {
	results := []Element{}
	for _, parent := range e.Children {
		results = append(results, parent)
		for _, child := range parent.Children {
			for _, childElement := range child.Flatten() {
				results = append(results, childElement)
			}
		}
	}
	return results
}

func (e Element) GetElementsByTagName(tagName string) []Element {
	tagNameLower := strings.ToLower(tagName)
	results := []Element{}
	for _, child := range e.Flatten() {
		if tagNameLower == strings.ToLower(child.ElementName) {
			results = append(results, child)
		}
	}
	return results
}

func (e Element) GetElementsByClassName(className string) []Element {
	classNameLower := strings.ToLower(className)
	results := []Element{}
	for _, child := range e.Flatten() {
		pieces := strings.Split(strings.ToLower(child.Attributes["class"]), " ")
		if sliceContains(pieces, classNameLower) {
			results = append(results, child)
		}
	}
	return results
}

func (e Element) GetElementById(id string) *Element {
	for _, child := range e.Flatten() {
		if child.Attributes["id"] == id {
			return &child
		}
	}
	return nil
}

func (e Element) QueryXpath(xpathQuery string) ([]Element, error) {
	//lex xpath query
	//chomp each token
	//build into result list
	//... profit?
	return []Element{}, nil
}

func (e Element) QuerySelector(cssSelectorQuery string) ([]Element, error) {
	return []Element{}, nil
}

func (e Element) GetText() string {
	textElements := e.GetElementsByTagName("text")
	textElementBodies := []string{}
	for _, textElement := range textElements {
		textElementBodies = append(textElementBodies, textElement.InnerHTML)
	}
	return strings.Join(textElementBodies, EMPTY)
}

func (e Element) EqualTo(e2 Element) bool {
	if e.ElementName != e2.ElementName {
		return false
	}
	if e.IsVoid != e2.IsVoid {
		return false
	}
	if e.IsClose != e2.IsClose {
		return false
	}
	if e.InnerHTML != e2.InnerHTML {
		return false
	}
	if len(e.Children) != len(e2.Children) {
		return false
	}
	if len(e.Attributes) != len(e2.Attributes) {
		return false
	}
	if !reflect.DeepEqual(e.Attributes, e2.Attributes) {
		return false
	}
	for index := 0; index < len(e.Children); index++ {
		childA := e.Children[index]
		childB := e2.Children[index]
		if !childA.EqualTo(childB) {
			return false
		}
	}

	return true
}

func (e Element) ToString() string {
	if e.IsRoot {
		return EMPTY
	}

	if e.IsText && isContinuousWhitespace([]rune(e.InnerHTML)) {
		return EMPTY
	} else if e.IsText {
		return trimString(e.InnerHTML)
	}

	if e.IsComment {
		return fmt.Sprintf("<!--%s-->", trimString(e.InnerHTML))
	}

	if e.IsVoid {
		if len(e.Attributes) == 0 {
			return fmt.Sprintf("<%s/>", e.ElementName)
		} else {
			return fmt.Sprintf("<%s %s/>", e.ElementName, stringifyMap(e.Attributes))
		}
	} else {
		if len(e.Attributes) == 0 {
			return fmt.Sprintf("<%s>", e.ElementName)
		} else {
			return fmt.Sprintf("<%s %s>", e.ElementName, stringifyMap(e.Attributes))
		}
	}

	return EMPTY
}

func (e Element) NonTextChildren() []Element {
	elems := []Element{}
	for _, c := range e.Children {
		elems = append(elems, c)
	}
	return elems
}

func (e Element) Render() string {
	if e.IsRoot {
		str := EMPTY
		for _, child := range e.Children {
			str = str + child.renderImpl(0)
		}
		return str
	} else {
		return e.renderImpl(0)
	}
}

func (e Element) renderImpl(nesting int) string {

	str := tabSequence(nesting) + e.ToString()

	str = str + "\n"

	for _, child := range e.Children {
		str = str + child.renderImpl(nesting+1)
	}

	if !(e.IsVoid || e.IsText || e.IsComment || e.IsRoot) {
		str = str + tabSequence(nesting) + fmt.Sprintf("</%s>\n", e.ElementName)
	}

	return str
}

//--------------------------------------------------------------------------------
// TYPES: ELEMENT STACK
//--------------------------------------------------------------------------------

type elementStackNode struct {
	Next  *elementStackNode
	Value Element
}

type elementStack struct {
	Top   *elementStackNode
	Count int
}

func (es *elementStack) Push(e Element) {
	es.Count = es.Count + 1
	if es.Top == nil {
		es.Top = &elementStackNode{Next: nil, Value: e}
	} else {
		oldTop := es.Top
		es.Top = &elementStackNode{Next: oldTop, Value: e}
	}
}

func (es *elementStack) Pop() *Element {
	if es.Top == nil {
		return nil
	}

	es.Count = es.Count - 1

	toReturn := es.Top.Value
	newNext := es.Top.Next
	es.Top = newNext
	return &toReturn
}

func (es elementStack) Peek() *Element {
	if es.Top == nil {
		return nil
	}
	return &es.Top.Value
}

func (es *elementStack) ToString() string {
	if es.Top == nil {
		return "*"
	}

	names := []string{}
	nodePtr := es.Top
	for nodePtr != nil {
		names = append([]string{nodePtr.Value.ElementName}, names...)
		nodePtr = nodePtr.Next
	}

	return strings.Join(names, " > ")
}

func (es elementStack) Duplicate() *elementStack {
	new_es := &elementStack{}

	if es.Top == nil {
		return new_es
	}

	nodes := []Element{}

	nodePtr := es.Top
	for nodePtr != nil {
		nodes = append([]Element{nodePtr.Value}, nodes...)
		nodePtr = nodePtr.Next
	}

	for _, node := range nodes {
		new_es.Push(node)
	}

	return new_es
}

//--------------------------------------------------------------------------------
// UTILITY
//--------------------------------------------------------------------------------

func readWhitespace(text []rune, cursor *int) ([]rune, error) {
	startingPosition := *cursor
	for ; *cursor < len(text); *cursor++ {
		c := text[*cursor]
		if !isWhitespace(c) {
			return text[startingPosition:*cursor], nil
		}
	}
	return text[startingPosition:*cursor], nil
}

func readUntilTag(text []rune, cursor *int) ([]rune, error) {
	startingPosition := *cursor
	for ; *cursor < len(text); *cursor++ {
		c := text[*cursor]
		if !isWhitespace(c) {
			if c == '<' {
				return text[startingPosition:*cursor], nil
			}
		}
	}
	return text[startingPosition:*cursor], nil
}

func readUntilScriptTagClose(text []rune, cursor *int, scriptType string) ([]rune, error) {
	starting_position := *cursor
	tag_start := 0
	working_tag := EMPTY

	const quote_double = rune('"')
	const quote_single = rune('\'')

	var quote_character rune

	state := 0
	for ; *cursor < len(text); *cursor++ {
		c := text[*cursor]

		switch state {
		case 0:
			if c == '/' && scriptType == "text/javascript" { //only kick off javascript style quote escapes if we're in js
				state = 21
			} else if c == '<' {
				tag_start = *cursor
				state = 11
			} else if c == quote_double || c == quote_single {
				state = 30
				quote_character = c
			}
			break
		case 11: //we're within a html tag in the code ...
			if c == '/' {
				state = 12
			}
			break
		case 12:
			if c == '>' {
				if strings.ToLower(working_tag) == "script" {
					*cursor = *cursor + 1
					return text[starting_position:tag_start], nil
				}
			} else if !isWhitespace(c) {
				working_tag = working_tag + string(c)
			}
			break
		case 21: //we hit a slash, which might be a comment
			if c == '*' {
				state = 25
			} else if c == '/' {
				state = 22
			} else {
				state = 0
			}
			break
		case 22: //read comment until newline or end of tag
			if c == '\n' {
				state = 0
			}
			break
		case 25: //almost a block comment close
			if c == '*' {
				state = 26
			}
		case 26: //definitely a block comment close
			if c == '/' {
				state = 0
			}
		case 30:
			if c == quote_character {
				state = 0
			}
		}
	}

	return text[starting_position:*cursor], nil
}

func readTag(text []rune, cursor *int) (*Element, error) {
	elem := Element{}

	state := 0

	attr_name := EMPTY
	attr_value := EMPTY
	const quote_double = rune('"')
	const quote_single = rune('\'')

	var quote_character rune

	for ; *cursor < len(text); *cursor++ {
		c := text[*cursor]
		switch state {
		case 0: //read until tag begins
			if c == '<' {
				state = 1
			}
			break
		case 1: //read preamble if any
			if c == '!' {
				elem.IsVoid = true
				state = 3
				continue
			} else if c == '/' {
				elem.IsClose = true
			} else if c == '>' {
				*cursor = *cursor + 1
				return &elem, errors.New("Empty tag similar to `<>` or `< >` or `</>`")
			} else if !isWhitespace(c) {
				state = 10
				elem.ElementName = elem.ElementName + string(c)
			} //else is whitespace, keep going
			break
		case 2: //read until end of tag
			if c == '/' {
				elem.IsVoid = true
			} else if c == '>' {
				elem.IsVoid = elem.IsVoid || isKnownVoidElement(elem.ElementName)
				*cursor = *cursor + 1
				return &elem, nil
			} else if !isWhitespace(c) {
				state = 100
				elem.ElementName = elem.ElementName + string(c)
			}
			break
		case 3: //possible xml comment
			if c == '-' {
				state = 4
			} else {
				*cursor = *cursor - 1
				state = 1
			}
			break
		case 4:
			if c == '-' {
				elem.IsComment = true
				state = 200 //consume xml comment
			} else {
				*cursor = *cursor + 1
				return &elem, errors.New("Almost an XML comment but not quite.")
			}
			break
		case 10: //read elemName
			if isWhitespace(c) {
				state = 20
			} else if c == '-' {
				state = 11
			} else if c == '>' {
				elem.IsVoid = elem.IsVoid || isKnownVoidElement(elem.ElementName)
				*cursor = *cursor + 1
				return &elem, nil
			} else if c == '/' {
				elem.IsVoid = true
				*cursor = *cursor + 1
				return &elem, nil
			} else {
				elem.ElementName = elem.ElementName + string(c)
			}
			break
		case 20: //read until attribute or end of tag
			if c == '/' {
				elem.IsVoid = true
				*cursor = *cursor + 1
				return &elem, nil
			} else if c == '>' {
				elem.IsVoid = elem.IsVoid || isKnownVoidElement(elem.ElementName)
				*cursor = *cursor + 1
				return &elem, nil
			} else if !isWhitespace(c) {
				*cursor = *cursor - 1
				state = 100
			}
			break
		case 100: //read attribute name
			if c == '=' {
				state = 101
			} else if c == '>' || c == '/' {
				if elem.Attributes == nil {
					elem.Attributes = map[string]string{}
				}
				elem.Attributes[attr_name] = ""
				*cursor = *cursor - 1
				state = 20
			} else if isWhitespace(c) {
				if elem.Attributes == nil {
					elem.Attributes = map[string]string{}
				}
				elem.Attributes[strings.ToLower(attr_name)] = ""
				attr_name = ""
				attr_value = ""
				state = 20
			} else {
				attr_name = attr_name + string(c)
			}
			break
		case 101: //set attribute value quote
			if c == quote_single || c == quote_double {
				quote_character = c
				state = 102
			} else if !isWhitespace(c) {
				attr_value = attr_value + string(c)
				state = 102
			}
		case 102: //read attribute value
			if isWhitespace(c) || c == quote_character {
				if elem.Attributes == nil {
					elem.Attributes = map[string]string{}
				}
				elem.Attributes[strings.ToLower(attr_name)] = attr_value
				attr_name = ""
				attr_value = ""
				state = 20
			} else {
				attr_value = attr_value + string(c)
			}
			break
		case 200:
			elem.ElementName = "XML COMMENT"
			elem.IsComment = true
			if c == '-' {
				state = 201
			} else {
				elem.InnerHTML = elem.InnerHTML + string(c)
			}
			break
		case 201:
			if c == '-' {
				state = 202
			} else {
				state = 200
				elem.InnerHTML = elem.InnerHTML + string(c)
			}
			break
		case 202:
			if c == '>' {
				*cursor = *cursor + 1
				return &elem, nil
			}
			break
		}
	}

	elem.IsVoid = elem.IsVoid || isKnownVoidElement(elem.ElementName)
	return &elem, nil
}

func sliceContains(slice []string, value string) bool {
	for _, e := range slice {
		if e == value {
			return true
		}
	}
	return false
}

func tabSequence(ofLength int) string {
	tabs := EMPTY
	for i := 0; i < ofLength; i++ {
		tabs = tabs + "  "
	}
	return tabs
}

func isKnownVoidElement(elementName string) bool {
	_, ok := KNOWN_VOID_ELEMENTS[strings.ToLower(elementName)]
	return ok
}

func stringifyMap(attributes map[string]string) string {
	pairs := []string{}
	for key, value := range attributes {
		if len(value) == 0 {
			pairs = append(pairs, key)
		} else {
			pairs = append(pairs, fmt.Sprintf("%s=\"%s\"", key, value))
		}
	}
	return strings.Join(pairs, " ")
}

func isContinuousWhitespace(corpus []rune) bool {
	for i := 0; i < len(corpus); i++ {
		c := corpus[i]
		if !isWhitespace(c) {
			return false
		}
	}
	return true
}

func trimString(text string) string {
	return string(trim([]rune(text)))
}

func trim(text []rune) []rune {
	if len(text) == 0 {
		return text
	}

	left := 0
	for ; left < len(text); left++ {
		c := text[left]
		if !isWhitespace(c) {
			break
		}
	}
	right := len(text) - 1
	for ; right > 0; right-- {
		c := text[right]
		if !isWhitespace(c) {
			break
		}
	}
	return text[left : right+1]
}

func isWhitespace(char rune) bool {
	switch char {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}
