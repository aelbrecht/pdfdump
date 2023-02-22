package main

import (
	"fmt"
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

	fmt.Printf("comparing %d with %d objects\n", len(first.Objects), len(second.Objects))

	pdf.HideIdentifiers = true
	pdf.HideVariableData = true
	pdf.HideRandomKeys = true

	bestMatches := make(map[string]string)
	bestMatchScores := make(map[string]float64)
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
				bestMatchScores[k1] = score
				break
			}
		}
	}

	iteration := 0
	for len(firstResolved) != len(first.Objects) || len(secondResolved) != len(second.Objects) {
		firstMatch := make(map[string]float64)
		secondMatch := make(map[string]float64)
		localMatches := make(map[string]string)

		for k1, o1 := range first.Objects {

			// Skip perfect matched objects
			if firstResolved[k1] {
				continue
			}

			bestScore := 0.0
			bestKey := ""
			for k2, o2 := range second.Objects {

				// Skip perfect matched objects
				if secondResolved[k2] {
					continue
				}

				// Calculate match
				score := pdf.MatchTypes(o1, o2)
				if score < 0.1 {
					continue
				}

				// Select best candidate based on distance
				if score > bestScore {
					bestScore = score
					bestKey = k2
				}
			}

			if bestKey != "" {
				if firstMatch[k1] < bestScore {
					localMatches[k1] = bestKey
				}
				firstMatch[k1] = math.Max(firstMatch[k1], bestScore)
				secondMatch[bestKey] = math.Max(secondMatch[bestKey], bestScore)
			}
		}

		if len(localMatches) == 0 {
			break
		}

		for k1, k2 := range localMatches {
			if firstMatch[k1] == secondMatch[k2] {
				bestMatches[k1] = k2
				bestMatchScores[k1] = firstMatch[k1]
				firstResolved[k1] = true
				secondResolved[k2] = true
			}
		}

		iteration++
		if iteration > 100 {
			log.Fatalln("infinite loop detected")
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

	index := 0
	for k1, k2 := range bestMatches {
		score := int(math.Round(bestMatchScores[k1] * 100))
		_, _ = f1.WriteString(fmt.Sprintf("# Object (%d) (%d%%)\n", index, score))
		_, _ = f2.WriteString(fmt.Sprintf("# Object (%d) (%d%%)\n", index, score))
		_, _ = f1.WriteString(first.Objects[k1].String())
		_, _ = f2.WriteString(second.Objects[k2].String())
		index++
	}

	for k, v := range first.Objects {
		if !firstResolved[k] {
			_, _ = f1.WriteString(fmt.Sprintf("# Object Unmatched\n"))
			_, _ = f1.WriteString(v.String())
		}
	}

	for k, v := range second.Objects {
		if !secondResolved[k] {
			_, _ = f2.WriteString(fmt.Sprintf("# Object Unmatched\n"))
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
