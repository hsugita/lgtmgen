/*
The MIT License (MIT)
Copyright (c) 2016 neko-neko.
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"flag"
	"io"
	"os"
	"image"
	"bytes"
	"sync"
	"fmt"
	"strings"
	"path/filepath"
	"io/ioutil"
	"github.com/disintegration/imaging"
	"github.com/neko-neko/lgtmgen/images"
)

// MaskImagePath
const MaskImage = "images/lgtm_mask.png"

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		output    string
		directory string
		force     bool

		version bool
	)

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(cli.errStream)

	flags.StringVar(&output, "output", "", "Output directory path")
	flags.StringVar(&output, "o", "", "Output directory path(Short)")

	flags.StringVar(&directory, "directory", "", "Input directory path")
	flags.StringVar(&directory, "d", "", "Input directory path(Short)")

	flags.BoolVar(&force, "force", false, "Force overwrite if outputfile exists")
	flags.BoolVar(&force, "f", false, "Force overwrite if outputfile exists(Short)")

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	// Parse commandline flag
	if err := flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(cli.errStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	// has targetDir?
	if directory == "" {
		fmt.Fprintf(cli.errStream, "input directory path is required.\n")
		return ExitCodeError
	}

	// has outputDir?
	if output == "" {
		fmt.Fprintf(cli.errStream, "output directory path is required.\n")
		return ExitCodeError
	}

	// add directory suffix
	directory = addDirectorySuffix(directory)
	output = addDirectorySuffix(output)

	// load mask image
	maskImage, err := loadMaskImage(MaskImage)
	if err != nil {
		fmt.Fprintf(cli.errStream, "fatal error %s.\n", err)
		return ExitCodeError
	}

	// load target images
	filePaths := readImagePaths(directory)

	// mask images
	wg := &sync.WaitGroup{}
	for _, filePath := range filePaths {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()

			maskedImage, mask_err := overlayImage(filePath, maskImage)
			if mask_err != nil {
				fmt.Fprintf(cli.errStream, "[%s] %s\n", mask_err, filePath)
				return
			}

			// generate output file path
			b := bytes.NewBuffer(make([]byte, 0))
			b.WriteString(output)
			b.WriteString(filepath.Base(filePath))
			outputFilePath := b.String()

			// save image file
			if existFile(outputFilePath) && !force {
				fmt.Fprintf(cli.errStream, "[already exists] %s\n", outputFilePath)
				return
			}
			save_err := imaging.Save(maskedImage, outputFilePath)
			if save_err != nil {
				fmt.Fprintf(cli.errStream, "[%s] %s\n", mask_err, filePath)
				return
			}
			fmt.Printf("[success] %s\n", outputFilePath)
		}(filePath)
	}
	wg.Wait()

	return ExitCodeOK
}

// Add directory suffix
// eg: directoryPath="/tmp" => directoryPath="/tmp/"
func addDirectorySuffix(directoryPath string) string {
	if strings.HasSuffix(directoryPath, "/") {
		return directoryPath
	}
	b := bytes.NewBuffer(make([]byte, 0))
	b.WriteString(directoryPath)
	b.WriteString("/")

	return b.String()
}

// Load mask image
func loadMaskImage(maskImage string) (image.Image, error) {
	imageByte, err := images.Asset(maskImage)
	if err != nil {
		return nil, err
	}

	// convert []byte to Image.image
	img, _, _ := image.Decode(bytes.NewReader(imageByte))
	return img, nil
}

// Get target image paths from target dir
func readImagePaths(target string) []string {
	files, err := ioutil.ReadDir(target)
	if err != nil {
		panic(err)
	}

	// create full path lists
	var filesPaths []string
	for _, fileInfo := range files {
		// skip directory
		if fileInfo.IsDir() {
			continue
		}

		b := bytes.NewBuffer(make([]byte, 0))
		b.WriteString(target)
		b.WriteString(fileInfo.Name())

		filesPaths = append(filesPaths, b.String())
	}

	return filesPaths
}

// Execute mask
func overlayImage(file string, maskImage image.Image) (*image.NRGBA, error) {
	srcImage, err := imaging.Open(file)
	if err != nil {
		return nil, err
	}

	maskedImage := imaging.OverlayCenter(srcImage, maskImage, 1.0)
	return maskedImage, nil
}

// Exists file
func existFile(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}