package main

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"testing"
)

func compare(t *testing.T, expected string, actual string) {
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
}

func TestNotFindTextToReplace(t *testing.T) {
	var origText = `
line 1
line 2
line 3
`
	var expected = `
line 1
line 2
line 3
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "pattern not found",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}

	if expected != replaceTextBetweenMarkers(origText, config) {
		t.Error("The pattern to replace should not have been found.")
	}
}

func TestFindOneMatchToReplace(t *testing.T) {
	var origText = `
line 1
line 2
line 3
# BEGIN MANAGED BLOCK
original block of text
# END MANAGED BLOCK
line 4
line 5
`
	var expected = `
line 1
line 2
line 3
# BEGIN MANAGED BLOCK
swapped with me
# END MANAGED BLOCK
line 4
line 5
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestNoIndentToWithIndent(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK
original block of text
# END MANAGED BLOCK
`
	var expected = `
line 1
line 2
    # BEGIN MANAGED BLOCK
    swapped with me
    # END MANAGED BLOCK
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       4,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestWithIndentToNoIndent(t *testing.T) {
	var origText = `
line 1
line 2
    # BEGIN MANAGED BLOCK
    original block of text
    # END MANAGED BLOCK
`
	var expected = `
line 1
line 2
# BEGIN MANAGED BLOCK
swapped with me
# END MANAGED BLOCK
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	var actual = replaceTextBetweenMarkers(origText, config)
	compare(t, expected, actual)
}

func TestFindMultipleMatchToReplace(t *testing.T) {
	var origText = `
line 1
# BEGIN MANAGED BLOCK
original block of text 1
# END MANAGED BLOCK
line 2
line 3
# BEGIN MANAGED BLOCK
original block of text 2
# END MANAGED BLOCK
line 4
line 5
`
	var expected = `
line 1
# BEGIN MANAGED BLOCK
swapped with me
# END MANAGED BLOCK
line 2
line 3
# BEGIN MANAGED BLOCK
swapped with me
# END MANAGED BLOCK
line 4
line 5
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestInsertBeforeExistingBlock(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK
original block of text
# END MANAGED BLOCK
line 3
`
	var expected = `
line 1
    # BEGIN MANAGED BLOCK
    swapped with me
    # END MANAGED BLOCK
line 2
line 3
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       4,
		Block:        "swapped with me",
		InsertBefore: "line 2",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestInsertAfterExistingBlock(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK
original block of text
# END MANAGED BLOCK
line 3
`
	var expected = `
line 1
    # BEGIN MANAGED BLOCK
    swapped with me
    # END MANAGED BLOCK
line 2
line 3
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       4,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "line 1",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestInsertAfterNoExistingBlock(t *testing.T) {
	var origText = `
line 1
line 2
line 3
`
	var expected = `
line 1
line 2
line 3
    # BEGIN MANAGED BLOCK
    swapped with me
    # END MANAGED BLOCK
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       4,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "line 3",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestInsertAfterNoExistingBlockNoMatchInsertAfter(t *testing.T) {
	var origText = `
line 1
line 2
line 3
`
	var expected = `
line 1
line 2
line 3
# BEGIN MANAGED BLOCK
swapped with me
# END MANAGED BLOCK
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "XXXX",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestStateIsFalseNoIndent(t *testing.T) {
	var origText = `
line 1
line 2
line 3
# BEGIN MANAGED BLOCK
swapped with me
# END MANAGED BLOCK
`
	var expected = `
line 1
line 2
line 3
`
	config := Config{
		Backup:       false,
		State:        false,
		Indent:       0,
		Block:        "swapped with me",
		InsertBefore: "XXXX",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestStateIsFalseWithIndent(t *testing.T) {
	var origText = `
line 1
line 2
line 3
      # BEGIN MANAGED BLOCK
      swapped with me
      # END MANAGED BLOCK
`
	var expected = `
line 1
line 2
line 3
`
	config := Config{
		Backup:       false,
		State:        false,
		Indent:       0,
		Block:        "swapped with me",
		InsertBefore: "XXXX",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}
