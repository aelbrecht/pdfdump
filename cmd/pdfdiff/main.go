package main

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"log"
	"math"
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

	dmp := diffmatchpatch.New()
	firstStrings := make(map[string]string, len(first.Objects))
	secondStrings := make(map[string]string, len(second.Objects))

	fmt.Printf("comparing %d with %d objects\n", len(first.Objects), len(second.Objects))

	pdf.HideIdentifiers = true
	pdf.HideVariableData = true
	for key, object := range first.Objects {
		firstStrings[key] = object.String()
	}
	for key, object := range second.Objects {
		secondStrings[key] = object.String()
	}
	bestMatches := make(map[string]string)

	firstResolved := make(map[string]bool)
	secondResolved := make(map[string]bool)

	for k1, s1 := range firstStrings {
		for k2, s2 := range secondStrings {

			// Skip perfect matched objects
			if secondResolved[k2] {
				continue
			}

			// Calculate diffs
			diffs := dmp.DiffMain(s1, s2, false)

			// Sum all distance scores
			dist := 0
			for _, diff := range diffs {
				if diff.Type != diffmatchpatch.DiffEqual {
					dist += len(diff.Text)
				}
			}

			// Lock perfect matches
			if dist == 0 {
				firstResolved[k1] = true
				secondResolved[k2] = true
				bestMatches[k1] = k2
				break
			}
		}
	}

	for k1, s1 := range firstStrings {

		// Skip perfect matched objects
		if firstResolved[k1] {
			continue
		}

		bestDist := math.MaxInt
		bestMatch := ""
		for k2, s2 := range secondStrings {

			// Skip perfect matched objects
			if secondResolved[k2] {
				continue
			}

			// Calculate diffs
			diffs := dmp.DiffMain(s1, s2, false)

			// Sum all distance scores
			dist := 0
			for _, diff := range diffs {
				if diff.Type != diffmatchpatch.DiffEqual {
					dist += len(diff.Text)
				}
			}
			maxLength := math.Max(float64(len(s1)), float64(len(s2)))
			matchRatio := 1 - (float64(dist) / maxLength)
			if matchRatio < 0.05 {
				continue
			}

			// Select best candidate based on distance
			if dist < bestDist {
				bestDist = dist
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
		_, _ = f1.WriteString(firstStrings[k1])
		_, _ = f2.WriteString(secondStrings[k2])
	}

	for k, v := range firstStrings {
		if !firstResolved[k] {
			_, _ = f1.WriteString(v)
			fmt.Println("add missing part for f1")
		}
	}

	for k, v := range secondStrings {
		if !secondResolved[k] {
			_, _ = f2.WriteString(v)
			fmt.Println("add missing part for f2")
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
