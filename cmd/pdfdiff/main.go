package main

import (
	"flag"
	"fmt"
	"github.com/aelbrecht/pdfdump/external/pdf"
	"log"
	"os"
	"path"
	"strings"
)

func writeDiffToDisk(diff *pdf.Comparison) error {
	f1, err := createOutputFile(diff.LeftPath)
	if err != nil {
		return err
	}
	_, _ = f1.WriteString(diff.LeftOutput)
	_ = f1.Close()

	f2, err := createOutputFile(diff.RightPath)
	if err != nil {
		return err
	}
	_, _ = f2.WriteString(diff.RightOutput)
	_ = f2.Close()

	return nil
}

func createOutputFile(filePath string) (*os.File, error) {
	dirName, fileName := path.Split(filePath)
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	return os.Create(path.Join(dirName, fileName+".txt"))
}

func printDivider(n int) {
	fmt.Print("\u001B[39m")
	for j := 0; j < n; j++ {
		fmt.Print("-")
	}
	fmt.Println()
}

func printDiff(result *pdf.Comparison, printAll bool) {
	difference := result.String()
	if printAll {
		fmt.Println(difference)
		return
	}

	lines := strings.Split(difference, "\n")
	maxLineLength := 0
	for _, line := range lines {
		tabs := strings.Count(line, "\t")
		length := len(line) - tabs + 8*tabs
		if length > maxLineLength && length < 300 {
			maxLineLength = length
		}
	}

	showDivider := false
	printDivider(maxLineLength)
	for i := 0; i < len(lines); i++ {
		if len(lines[i]) == 0 {
			continue
		}
		if lines[i][0] == '+' {
			fmt.Printf("\033[92m%s\n", lines[i])
			showDivider = true
			continue
		} else if lines[i][0] == '-' {
			fmt.Printf("\033[91m%s\n", lines[i])
			showDivider = true
			continue
		}

		isVisible := false
		for j := -5; j < 6; j++ {
			index := i + j
			if index < 0 || index >= len(lines) || len(lines[index]) == 0 {
				continue
			}
			c := lines[index][0]
			if c == '+' || c == '-' {
				isVisible = true
				break
			}
		}
		if isVisible {
			fmt.Printf("\033[39m%s\n", lines[i])
		} else if showDivider {
			showDivider = false
			printDivider(maxLineLength)
		}
	}
	printDivider(maxLineLength)
}

func main() {

	shouldDump := flag.Bool("dump", false, "write the comparable text file to disk using")
	shouldDiff := flag.Bool("diff", false, "output diff to stdout")
	isVerbose := flag.Bool("verbose", false, "output stats")
	printAll := flag.Bool("full", false, "print full difference")
	leftPath := flag.String("left", "", "left input file")
	rightPath := flag.String("right", "", "right input file")
	flag.Parse()

	if *leftPath == "" || *rightPath == "" {
		log.Fatalln("error: no input files specified")
	}

	result := pdf.Compare(*leftPath, *rightPath, *isVerbose)
	hasAction := false
	if *shouldDiff {
		printDiff(result, *printAll)
		hasAction = true
	}
	if *shouldDump {
		err := writeDiffToDisk(result)
		if err != nil {
			log.Fatalln(err)
		}
		hasAction = true
	}

	if !hasAction {
		log.Fatalln("error: no action specified")
	}
}
