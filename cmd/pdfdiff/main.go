package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"pdfdump/internal/pdf"
	"pdfdump/internal/token"
	"strings"
)

func parsePDF(filePath string) *pdf.PDF {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}
	scanner := token.NewScanner(f)
	parser := pdf.NewParser(scanner)
	parser.Parse()
	_ = f.Close()
	return parser.PDF()
}

func diffPDF(firstPath string, secondPath string) {

	first := parsePDF(firstPath)
	second := parsePDF(secondPath)

	pdf.HideIdentifiers = true
	pdf.HideVariableData = true
	pdf.HideRandomKeys = true

	bestMatches := make(map[string]string)
	firstResolved := make(map[string]bool)
	secondResolved := make(map[string]bool)

	for k1, o1 := range first.Objects {
		for k2, o2 := range second.Objects {

			// Skip perfect matched objects
			if secondResolved[k2] {
				continue
			}

			// Calculate match
			score := pdf.MatchTypes(o1, o2)

			// Lock perfect matches
			if score == 1.0 {
				firstResolved[k1] = true
				secondResolved[k2] = true
				bestMatches[k1] = k2
				break
			}
		}
	}

	for k1, o1 := range first.Objects {

		// Skip perfect matched objects
		if firstResolved[k1] {
			continue
		}

		bestMatchScore := 0.0
		bestMatch := ""
		for k2, o2 := range second.Objects {

			// Skip perfect matched objects
			if secondResolved[k2] {
				continue
			}

			// Calculate match
			score := pdf.MatchTypes(o1, o2)

			// Select best candidate based on distance
			if score > bestMatchScore {
				bestMatchScore = score
				bestMatch = k2
			}
		}

		if bestMatch != "" {
			bestMatches[k1] = bestMatch
			firstResolved[k1] = true
			secondResolved[bestMatch] = true
		}
	}

	f1, err := createOutputFile(firstPath)
	if err != nil {
		log.Fatalln(err)
	}
	f2, err := createOutputFile(secondPath)
	if err != nil {
		log.Fatalln(err)
	}

	for k1, k2 := range bestMatches {
		_, _ = f1.WriteString(first.Objects[k1].String())
		_, _ = f2.WriteString(second.Objects[k2].String())
	}

	for k, v := range first.Objects {
		if !firstResolved[k] {
			_, _ = f1.WriteString(v.String())
		}
	}

	for k, v := range second.Objects {
		if !secondResolved[k] {
			_, _ = f2.WriteString(v.String())
		}
	}

	_ = f1.Close()
	_ = f2.Close()
}

func createOutputFile(filePath string) (*os.File, error) {
	dirName, fileName := path.Split(filePath)
	fileName = strings.TrimSuffix(fileName, path.Ext(fileName))
	return os.Create(path.Join(dirName, fileName+".txt"))
}

func main() {
	if len(os.Args) != 3 {
		log.Fatalln("expected 2 files as input args")
	}
	diffPDF(os.Args[1], os.Args[2])
}
