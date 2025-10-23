package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

type Config struct {
	Backup, State                                                  bool
	Indent                                                         int
	Block, InsertBefore, InsertAfter, BeginMarker, EndMarker, Path string
	Mode, Owner, Group                                             string
}

func main() {
	var indent int
	var backup, block, insertBefore, insertAfter, marker, markerBegin, markerEnd, path, state, mode, owner, group string

	flags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "backup",
			Usage:       "create a backup file including the timestamp information so you can get the original file back if you somehow clobbered it incorrectly.",
			Destination: &backup,
			DefaultText: "false",
			Value:       "false",
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
			Usage:       "The file to modify. If the path is relative, the working directory of where blockinfile is running will be pre-fixed to the path.",
			Destination: &path,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "state",
			Usage:       "Whether the block should be there or not.",
			Destination: &state,
			DefaultText: "true",
			Value:       "true",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "mode",
			Usage:       "The permissions the resulting file should have. For example, '0644' or '0755'.",
			Destination: &mode,
			Value:       "",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "owner",
			Usage:       "Name of the user that should own the file.",
			Destination: &owner,
			Value:       "",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:        "group",
			Usage:       "Name of the group that should own the file.",
			Destination: &group,
			Value:       "",
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
		Version: "v0.1.10",
		Action: func(c *cli.Context) error {
			var backupAsBool, _ = strconv.ParseBool(backup)
			var stateAsBool, _ = strconv.ParseBool(state)
			config := Config{
				Backup:       backupAsBool,
				State:        stateAsBool,
				Indent:       indent,
				Block:        block,
				InsertBefore: insertBefore,
				InsertAfter:  insertAfter,
				BeginMarker:  strings.Replace(marker, "{mark}", markerBegin, 1),
				EndMarker:    strings.Replace(marker, "{mark}", markerEnd, 1),
				Path:         getFullPath(path),
				Mode:         mode,
				Owner:        owner,
				Group:        group,
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

// If path is relative, add working directory as prefix to path; otherwise, return the existing full path
func getFullPath(path string) string {
	if strings.HasPrefix(path, string(os.PathSeparator)) {
		return path
	}
	wd, _ := os.Getwd()
	return wd + string(os.PathSeparator) + path
}

func replaceTextBetweenMarkersInFile(config Config) {
	// Read entire file content, giving us little control but
	// making it very simple. No need to close the file.
	content, err := ioutil.ReadFile(config.Path)
	if err != nil {
		log.Fatal(err)
	}

	updatedContent := replaceTextBetweenMarkers(string(content), config)
	if string(content) != updatedContent {
		if config.Backup {
			backupFile(config.Path)
		}

		f, err := os.OpenFile(config.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Fatal(err)
		}
		_, err = f.WriteString(updatedContent)
		defer f.Close()
	}
}

func removeExistingBlock(sourceText, beginMarker, endMarker string) string {
	// Add \n because markers could have similar prefix, \n will make sure match to end of line
	beginIndex := strings.LastIndex(sourceText, beginMarker+"\n")
	if beginIndex >= 0 {
		sourceText = removeLeadingSpacesOfBlock(sourceText, beginIndex)
		// After removing leading spaces, reset beginIndex
		beginIndex := strings.LastIndex(sourceText, beginMarker+"\n")

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
		reAddSpaces.ReplaceAllLiteralString(config.Block, "\n"+strings.Repeat(" ", config.Indent)))

	switch {
	case !config.State:
		return removeExistingBlock(sourceText, config.BeginMarker, config.EndMarker)
	case config.InsertBefore != "":
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
	case config.InsertAfter != "":
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
	case strings.Contains(sourceText, config.BeginMarker+"\n"):
		// Remove any leading spaces before replacing the block in case indentation changed
		beginIndex := strings.LastIndex(sourceText, config.BeginMarker+"\n")
		sourceText = removeLeadingSpacesOfBlock(sourceText, beginIndex)

		// Replace existing block
		reReplaceMarker := regexp.MustCompile(fmt.Sprintf("(?s)%s(.*?)%s",
			regexp.QuoteMeta(config.BeginMarker)+"\n", regexp.QuoteMeta(config.EndMarker)))
		return reReplaceMarker.ReplaceAllLiteralString(sourceText,
			fmt.Sprintf("%s\n%s\n%s",
				paddedBeginMarker,
				paddedReplaceText,
				paddedEndMarker),
		)
	default:
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

	replaceTextBetweenMarkersInFile(config)
	
	// Apply ownership and permissions after file modification
	if err := applyFileAttributes(config); err != nil {
		log.Fatal(err)
	}
}

// applyFileAttributes applies mode, owner, and group settings to the file
func applyFileAttributes(config Config) error {
	// Apply owner and group
	if config.Owner != "" || config.Group != "" {
		if err := applyOwnership(config.Path, config.Owner, config.Group); err != nil {
			return err
		}
	}
	
	// Apply mode (permissions)
	if config.Mode != "" {
		if err := applyMode(config.Path, config.Mode); err != nil {
			return err
		}
	}
	
	return nil
}

// applyOwnership changes the owner and/or group of the file
func applyOwnership(path, owner, group string) error {
	// Build chown command
	var chownArg string
	
	if owner != "" && group != "" {
		chownArg = owner + ":" + group
	} else if owner != "" {
		chownArg = owner
	} else if group != "" {
		chownArg = ":" + group
	} else {
		return nil // Nothing to do
	}
	
	cmd := exec.Command("chown", chownArg, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to change ownership: %s, error: %w", string(output), err)
	}
	
	return nil
}

// applyMode changes the file permissions
func applyMode(path, mode string) error {
	// Parse mode string to os.FileMode
	// Handle octal mode (e.g., "0644", "644")
	modeStr := strings.TrimPrefix(mode, "0")
	modeInt, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		// If parsing as octal fails, try symbolic mode via chmod command
		return applyModeViaChmod(path, mode)
	}
	
	fileMode := os.FileMode(modeInt)
	if err := os.Chmod(path, fileMode); err != nil {
		return fmt.Errorf("failed to change mode: %w", err)
	}
	
	return nil
}

// applyModeViaChmod uses the chmod command for symbolic modes (e.g., "u+rwx")
func applyModeViaChmod(path, mode string) error {
	cmd := exec.Command("chmod", mode, path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to change mode via chmod: %s, error: %w", string(output), err)
	}
	return nil
}

// Does not update the file’s modification timestamp.
func touchFile(path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}
