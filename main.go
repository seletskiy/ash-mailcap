package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"text/template"

	"github.com/docopt/docopt-go"
)

var defaultEditorWrapper = `#!/bin/sh

exec vim "+/\[{{.CommentID}}@[0-9]\+\]" "${@}"
`

var usage = `ash mailcap helper.

Program expects to get e-mail which Stash sends when someone adds comment to
the review.

This program will replace $EDITOR environment variable when launching ash,
so it will point to the specially generated executable wrapper, which will
open editor and scroll to the comment that is mentioned in the <file>.

Default wrapper for editor:

    ` + strings.Replace(defaultEditorWrapper, "\n", "\n    ", -1) + `
Usage:
  $0 -h | --help
  $0 [options] <file>

Options:
  -h --help      Show this help message.
  -t <template>  Path to file, that will be used as template file for mocking
                 editor.
                 Example for this file is given above.
                 If file is not specified, default value will be used.
                 File should be written using golang template syntax, and two
                 variable will be passed:
                     * .CommentID - comment, mentioned in the <file>;
                     * .ReviewURL - full URL for the review;
  -x <action>    Action that will be executed if specified file does not
                 contains link to comment.
`

var reStashCommentLink = regexp.MustCompile(
	`http://[^/]+/(?:projects|users)/[^/]+/repos/[^/]+/` +
		`pull-requests/\d+/overview\?commentId=(\d+)`,
)

func main() {
	args, err := docopt.Parse(
		strings.Replace(usage, "$0", os.Args[0], -1), nil, true, "1.0", false,
	)

	if err != nil {
		panic(err)
	}

	inputFile, err := os.Open(args["<file>"].(string))
	if err != nil {
		log.Fatalf("can't open specified file: %s", err)
	}

	inputData, err := ioutil.ReadAll(inputFile)
	if err != nil {
		log.Fatalf("can't read specified file: %s", err)
	}

	editorTemplate := template.Must(
		template.New("").Parse(defaultEditorWrapper),
	)

	if args["-t"] != nil {
		editorTemplate = template.Must(
			template.ParseFiles(args["-t"].(string)),
		)
	}

	matches := reStashCommentLink.FindStringSubmatch(string(inputData))
	if len(matches) != 0 {
		var (
			reviewURL = matches[0]
			commentID = matches[1]
		)

		openTempEditor(editorTemplate, reviewURL, commentID)
		os.Exit(0)
	}

	if args["-x"] == nil {
		log.Printf("specified file does not contains link to comment")
		os.Exit(1)
	}

	openExternalProgram(args["-x"].(string))
}

func openExternalProgram(cmdline string) {
	externalProgram := exec.Command(
		"/bin/sh", "-c", cmdline,
	)

	externalProgram.Stdin = os.Stdin
	externalProgram.Stdout = os.Stdout
	externalProgram.Stderr = os.Stderr

	err := externalProgram.Start()
	if err != nil {
		log.Printf("can't start external command: %s", err)
		os.Exit(1)
	} else {
		err := externalProgram.Wait()
		if err != nil {
			log.Printf("error running external program: %s", err)
			os.Exit(3)
		}

		os.Exit(0)
	}
}

func openTempEditor(
	editorTemplate *template.Template,
	reviewURL string,
	commentID string,
) {
	hash := getHash(reviewURL, commentID)

	cachePath := filepath.Join(os.TempDir(), "ash-mailcap-cache."+hash)
	cacheFile, err := os.OpenFile(cachePath, os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		log.Printf("can't open cache file %s: %s", cachePath, err)
		return
	}

	defer cacheFile.Close()

	cacheData, err := ioutil.ReadAll(cacheFile)
	if err != nil {
		log.Printf("can't read cache file %s: %s", cachePath, err)
		return
	}

	if len(cacheData) > 0 {
		fmt.Print(string(cacheData))
		return
	}

	tempEditorExecutable, err := ioutil.TempFile(
		os.TempDir(), "ash-mailcap-editor.",
	)
	if err != nil {
		log.Printf("can't create temporary file: %s", err)
		return
	}

	defer func() {
		os.Remove(tempEditorExecutable.Name())
	}()

	err = os.Chmod(tempEditorExecutable.Name(), 0777)
	if err != nil {
		log.Printf(
			"can't chmod +x <%s>: %s", tempEditorExecutable.Name(), err,
		)
		return
	}

	editorTemplate.Execute(tempEditorExecutable, map[string]string{
		"ReviewURL": reviewURL,
		"CommentID": commentID,
	})

	tempEditorExecutable.Close()

	ash := exec.Command("ash", reviewURL, "review")
	ash.Env = append(
		[]string{
			"EDITOR=" + tempEditorExecutable.Name(),
			"HOME=" + os.ExpandEnv("$HOME"),
		}, os.Environ()...,
	)

	ash.Stdin = os.Stdin

	ash.Stdout = io.MultiWriter(os.Stdout, cacheFile)
	ash.Stderr = io.MultiWriter(os.Stderr, cacheFile)

	err = ash.Run()
	if err != nil {
		switch ashError := err.(type) {
		case *exec.ExitError:
			if ashError.Sys().(syscall.WaitStatus).ExitStatus() != 2 {
				log.Printf("ash exited with: %s", err)
			}
		default:
			log.Printf("can't run ash: %s", err)
		}
	}
}

func getHash(identifiers ...string) string {
	hasher := md5.New()
	hasher.Write([]byte(strings.Join(identifiers, "__")))
	return hex.EncodeToString(hasher.Sum(nil))
}
