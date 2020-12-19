package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

type Config struct {
	Backup, State                                                  bool
	Indent                                                         int
	Block, InsertBefore, InsertAfter, BeginMarker, EndMarker, Path string
}

func main() {
	var backup, state bool
	var indent int
	var block, insertBefore, insertAfter, marker, markerBegin, markerEnd, path string

	flags := []cli.Flag{
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "backup",
			Usage:       "create a backup file including the timestamp information so you can get the original file back if you somehow clobbered it incorrectly.",
			Destination: &backup,
			DefaultText: "false",
			Value:       false,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name: "block",
			Usage: `The text to insert inside the marker lines.
					If it is missing or an empty string, the block will be removed as if state were specified to absent.`,
			Destination: &block,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:        "indent",
			Usage:       "The number of spaces to indent the block. Indent must be >= 0.",
			Destination: &indent,
			DefaultText: "0",
			Value:       0,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name: "insertafter",
			Usage: `If specified and no begin/ending marker lines are found, the block will be inserted after the last match of specified regular expression.
					A special value is available; EOF for inserting the block at the end of the file.
					If specified regular expression has no matches, EOF will be used instead.`,
			Destination: &insertAfter,
			Value:       "",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name: "insertbefore",
			Usage: `If specified and no begin/ending marker lines are found, the block will be inserted before the last match of specified regular expression.
					A special value is available; BOF for inserting the block at the beginning of the file.
				    If specified regular expression has no matches, the block will be inserted at the end of the file.`,
			Destination: &insertBefore,
			Value:       "",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name: "marker",
			Usage: `The marker line template.
				    {mark} will be replaced with the values in marker_begin (default="BEGIN") and marker_end (default="END").
				    Using a custom marker without the {mark} variable may result in the block being repeatedly inserted on subsequent playbook runs.`,
			Destination: &marker,
			Value:       "# {mark} MANAGED BLOCK",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "markerbegin",
			Usage:       "This will be inserted at {mark} in the opening ansible block marker.",
			Destination: &markerBegin,
			DefaultText: "BEGIN",
			Value:       "BEGIN",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "markerend",
			Usage:       "This will be inserted at {mark} in the closing ansible block marker.",
			Destination: &markerEnd,
			DefaultText: "END",
			Value:       "END",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "path",
			Usage:       "the file to modify",
			Destination: &path,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:        "state",
			Usage:       "Whether the block should be there or not.",
			Destination: &state,
			DefaultText: "true",
			Value:       true,
		}),
		&cli.StringFlag{
			Name:  "config",
			Usage: "YAML configuration file containing parameters for blockinfile",
		},
	}

	// TODO Dynamically set the Version
	app := &cli.App{
		Name:    "blockinfile",
		Usage:   "insert/update/remove a block of multi-line text surrounded by customizable marker lines",
		Version: "v0.0.6",
		Action: func(c *cli.Context) error {
			config := Config{
				Backup:       backup,
				State:        state,
				Indent:       indent,
				Block:        block,
				InsertBefore: insertBefore,
				InsertAfter:  insertAfter,
				BeginMarker:  strings.Replace(marker, "{mark}", markerBegin, 1),
				EndMarker:    strings.Replace(marker, "{mark}", markerEnd, 1),
				Path:         path,
			}

			updateBlockInFile(config)
			return nil
		},
		Before: altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("config")),
		Flags:  flags,
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func backupFile(sourceFile string) {
	input, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Fatal(err)
		return
	}

	var backupFile = sourceFile + "." + time.Now().Format(time.RFC3339)

	if err := ioutil.WriteFile(backupFile, input, 0644); err != nil {
		fmt.Println("Error creating", backupFile)
		log.Fatal(err)
		return
	}
}

func checkFlags(config Config) error {
	if config.Path == "" {
		return errors.New("required flag \"path\" not set")
	}
	if config.InsertBefore != "" && config.InsertAfter != "" {
		return errors.New("only one of these flags can be used at a time [markerbegin|markerend]")
	}
	return nil
}

func replaceTextBetweenMarkersInFile(config Config) {
	// Read entire file content, giving us little control but
	// making it very simple. No need to close the file.
	content, err := ioutil.ReadFile(config.Path)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(config.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(replaceTextBetweenMarkers(string(content), config))
	defer f.Close()
}

func removeExistingBlock(sourceText, beginMarker, endMarker string) string {
	beginIndex := strings.LastIndex(sourceText, beginMarker)
	if beginIndex >= 0 {
		sourceText = removeLeadingSpacesOfBlock(sourceText, beginIndex)
		// After removing leading spaces, reset beginIndex
		beginIndex := strings.LastIndex(sourceText, beginMarker)

		endIndex := strings.LastIndex(sourceText, endMarker) + len(endMarker) + 1
		return sourceText[:beginIndex] + sourceText[endIndex:]
	}
	return sourceText
}

func removeLeadingSpacesOfBlock(sourceText string, beginIndex int) string {
	// Remove any leading spaces of block
	beginNonSpaceIndex := beginIndex
	for nonSpaceIndex := beginIndex - 1; nonSpaceIndex >= 0; nonSpaceIndex-- {
		if sourceText[nonSpaceIndex] != ' ' {
			break
		}
		beginNonSpaceIndex = nonSpaceIndex
	}
	sourceText = sourceText[:beginNonSpaceIndex] + sourceText[beginIndex:]
	return sourceText
}

func replaceTextBetweenMarkers(sourceText string, config Config) string {
	reAddSpaces := regexp.MustCompile(`\r?\n`)
	paddedBeginMarker := fmt.Sprintf("%s%s", strings.Repeat(" ", config.Indent), config.BeginMarker)
	paddedEndMarker := fmt.Sprintf("%s%s", strings.Repeat(" ", config.Indent), config.EndMarker)
	paddedReplaceText := fmt.Sprintf("%s%s", strings.Repeat(" ", config.Indent),
		reAddSpaces.ReplaceAllString(config.Block, "\n"+strings.Repeat(" ", config.Indent)))

	if !config.State {
		// Remove the block
		return removeExistingBlock(sourceText, config.BeginMarker, config.EndMarker)
	} else if config.InsertBefore != "" {
		sourceText = removeExistingBlock(sourceText, config.BeginMarker, config.EndMarker)

		var index = strings.LastIndex(sourceText, config.InsertBefore)
		// Not found, insert at EOF
		if index < 0 {
			return fmt.Sprintf("%s%s\n%s\n%s\n",
				sourceText,
				paddedBeginMarker,
				paddedReplaceText,
				paddedEndMarker)
		}
		// Insert before
		return fmt.Sprintf("%s%s\n%s\n%s\n%s",
			sourceText[:index],
			paddedBeginMarker,
			paddedReplaceText,
			paddedEndMarker,
			sourceText[index:])
	} else if config.InsertAfter != "" {
		sourceText = removeExistingBlock(sourceText, config.BeginMarker, config.EndMarker)

		var index = strings.LastIndex(sourceText, config.InsertAfter)
		// Not found, insert at EOF
		if index < 0 {
			return fmt.Sprintf("%s%s\n%s\n%s\n",
				sourceText,
				paddedBeginMarker,
				paddedReplaceText,
				paddedEndMarker)
		}
		// Insert after
		index = index + len(config.InsertAfter)
		return fmt.Sprintf("%s\n%s\n%s\n%s%s",
			sourceText[:index],
			paddedBeginMarker,
			paddedReplaceText,
			paddedEndMarker,
			sourceText[index:])
	} else if strings.Contains(sourceText, config.BeginMarker) {
		// Remove any leading spaces before replacing the block in case indentation changed
		beginIndex := strings.LastIndex(sourceText, config.BeginMarker)
		sourceText = removeLeadingSpacesOfBlock(sourceText, beginIndex)

		// Replace existing block
		reReplaceMarker := regexp.MustCompile(fmt.Sprintf("(?s)%s(.*?)%s", config.BeginMarker, config.EndMarker))
		return reReplaceMarker.ReplaceAllString(sourceText,
			fmt.Sprintf("%s\n%s\n%s",
				paddedBeginMarker,
				paddedReplaceText,
				paddedEndMarker),
		)
	} else {
		// Not found, add to EOF
		return fmt.Sprintf("%s%s\n%s\n%s\n",
			sourceText,
			paddedBeginMarker,
			paddedReplaceText,
			paddedEndMarker)
	}
}

func updateBlockInFile(config Config) {
	if err := checkFlags(config); err != nil {
		log.Fatal(err)
	}

	// Make sure file exists by touching it
	if err := touchFile(config.Path); err != nil {
		log.Fatal(err)
	}

	if config.Backup {
		backupFile(config.Path)
	}

	replaceTextBetweenMarkersInFile(config)
}

func touchFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}
