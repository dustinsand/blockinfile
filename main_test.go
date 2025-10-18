package main

import (
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"
)

func compare(t *testing.T, expected string, actual string) {
	if expected != actual {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(actual, expected, false)
		t.Errorf("The differences are:%s", dmp.DiffPrettyText(diffs))
	}
}

// getModTimeFromFile returns the modification time of an already opened file.
func getModTimeFromFile(file *os.File) (time.Time, error) {
	info, err := file.Stat()
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func TestEmptyFile(t *testing.T) {
	var origText = ""
	var expected = `# BEGIN MANAGED BLOCK
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

	if expected != replaceTextBetweenMarkers(origText, config) {
		t.Error("block should have been added to EOF")
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
# BEGIN MANAGED BLOCK
pattern not exist before
# END MANAGED BLOCK
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "pattern not exist before",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}

	compare(t, expected, replaceTextBetweenMarkers(origText, config))
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

func TestDollarSignToReplace(t *testing.T) {
	var origText = `
line 1
$USER
$20
line 2
line 3
#!/bin/bash
$USER1 original block of text $VAR1
# managed file end
line 4
line 5
`
	var expected = `
line 1
$USER
$20
line 2
line 3
#!/bin/bash
$1 swapped with $VAR2 lorem ipsum. $$$ lorem ipsum. Echo $USER
# managed file end
line 4
line 5
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "$1 swapped with $VAR2 lorem ipsum. $$$ lorem ipsum. Echo $USER",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "#!/bin/bash",
		EndMarker:    "# managed file end",
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

func TestInsertBlockSimilarPrefixMarker(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
`
	var expected = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
# BEGIN MANAGED BLOCK - Common
new block
# END MANAGED BLOCK - Common
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "new block",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK - Common",
		EndMarker:    "# END MANAGED BLOCK - Common",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestReplaceBlockSimilarPrefixMarker(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
# BEGIN MANAGED BLOCK - Common
new block
# END MANAGED BLOCK - Common
`
	var expected = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
# BEGIN MANAGED BLOCK - Common
replaced similar prefix block
# END MANAGED BLOCK - Common
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       0,
		Block:        "replaced similar prefix block",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK - Common",
		EndMarker:    "# END MANAGED BLOCK - Common",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestRemoveBlockSimilarPrefixMarker(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
# BEGIN MANAGED BLOCK - Common
new block
# END MANAGED BLOCK - Common
`
	var expected = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
`
	config := Config{
		Backup:       false,
		State:        false,
		Indent:       0,
		Block:        "new block",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK - Common",
		EndMarker:    "# END MANAGED BLOCK - Common",
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

func TestInsertBeforeNonExistingBlock(t *testing.T) {
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
		InsertBefore: "i do not exist",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestInsertBeforeBlockSimilarPrefixMarker(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
`
	var expected = `
    # BEGIN MANAGED BLOCK - Common
    new block
    # END MANAGED BLOCK - Common
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       4,
		Block:        "new block",
		InsertBefore: "line 1",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK - Common",
		EndMarker:    "# END MANAGED BLOCK - Common",
		Path:         "",
	}
	compare(t, expected, replaceTextBetweenMarkers(origText, config))
}

func TestInsertAfterBlockSimilarPrefixMarker(t *testing.T) {
	var origText = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
`
	var expected = `
line 1
line 2
# BEGIN MANAGED BLOCK - Common Global
original block of text
# END MANAGED BLOCK - Common Global
line 3
    # BEGIN MANAGED BLOCK - Common
    new block
    # END MANAGED BLOCK - Common
`
	config := Config{
		Backup:       false,
		State:        true,
		Indent:       4,
		Block:        "new block",
		InsertBefore: "",
		InsertAfter:  "line 3",
		BeginMarker:  "# BEGIN MANAGED BLOCK - Common",
		EndMarker:    "# END MANAGED BLOCK - Common",
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

func TestExistingFileAddBlock(t *testing.T) {
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
	f, err := ioutil.TempFile("", "sample")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(origText)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	config := Config{
		Backup:       false,
		State:        true,
		Indent:       6,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         f.Name(),
	}
	fBeforeModTime, err := getModTimeFromFile(f)
	updateBlockInFile(config)
	fAfterModTime, err := getModTimeFromFile(f)
	if fBeforeModTime.UnixNano() > fAfterModTime.UnixNano() {
		log.Fatal("Expected fAfterModTime to be after fBeforeModTime")
	}

	actual, err := ioutil.ReadFile(config.Path)
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(f.Name())

	compare(t, expected, string(actual))
}

func TestFileNotExistAddBlock(t *testing.T) {
	var expected = `      # BEGIN MANAGED BLOCK
      swapped with me
      # END MANAGED BLOCK
`
	f, err := ioutil.TempFile("", "sample")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	config := Config{
		Backup:       false,
		State:        true,
		Indent:       6,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         f.Name(),
	}
	fBeforeModTime, err := getModTimeFromFile(f)
	updateBlockInFile(config)
	fAfterModTime, err := getModTimeFromFile(f)
	if fBeforeModTime.UnixNano() > fAfterModTime.UnixNano() {
		log.Fatal("Expected fAfterModTime to be after fBeforeModTime")
	}
	updateBlockInFile(config)

	actual, err := ioutil.ReadFile(config.Path)
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(f.Name())

	compare(t, expected, string(actual))
}

func TestNoChange(t *testing.T) {
	var origText = `
      # BEGIN MANAGED BLOCK
      swapped with me
      # END MANAGED BLOCK
`
	var expected = `
      # BEGIN MANAGED BLOCK
      swapped with me
      # END MANAGED BLOCK
`
	f, err := ioutil.TempFile("", "sample")
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(origText)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	config := Config{
		Backup:       false,
		State:        true,
		Indent:       6,
		Block:        "swapped with me",
		InsertBefore: "",
		InsertAfter:  "",
		BeginMarker:  "# BEGIN MANAGED BLOCK",
		EndMarker:    "# END MANAGED BLOCK",
		Path:         f.Name(),
	}
	fBeforeModTime, err := getModTimeFromFile(f)
	updateBlockInFile(config)
	fAfterModTime, err := getModTimeFromFile(f)
	if fBeforeModTime.UnixNano() != fAfterModTime.UnixNano() {
		log.Fatal("File was not updated so expected fAfterModTime to be equal to fBeforeModTime")
	}

	actual, err := ioutil.ReadFile(config.Path)
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(f.Name())

	compare(t, expected, string(actual))
}

func TestGetFullPath(t *testing.T) {
	wd, _ := os.Getwd()

	assert.Equal(t, wd, getFullPath(wd))
	assert.Equal(t, wd+"/../foo/bar", getFullPath("../foo/bar"))
	assert.Equal(t, wd+"/./foo/bar", getFullPath("./foo/bar"))
	assert.Equal(t, wd+"/foo/bar", getFullPath("foo/bar"))
	assert.Equal(t, "/fullpath/foo/bar", getFullPath("/fullpath/foo/bar"))
}
