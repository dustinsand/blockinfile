package main

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"testing"
)

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
	if expected != replaceTextBetweenMarkers(origText,
		"pattern not found",
		"BEGIN",
		"END",
		"",
		"",
		0,
		true) {
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"",
		"",
		0,
		true)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
}

func TestIndent(t *testing.T) {
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"",
		"",
		4,
		true)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"",
		"",
		0,
		true)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"line 2",
		"",
		4,
		true)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"",
		"line 1",
		4,
		true)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"",
		"line 3",
		4,
		true)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"",
		"XXXX",
		0,
		true)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
}

func TestStateIsFalse(t *testing.T) {
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
	var actual = replaceTextBetweenMarkers(origText,
		"swapped with me",
		"# BEGIN MANAGED BLOCK",
		"# END MANAGED BLOCK",
		"XXXX",
		"",
		0,
		false)
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Error("The differences are:")
		fmt.Println(dmp.DiffPrettyText(diffs))
	}
}