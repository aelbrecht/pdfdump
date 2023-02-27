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

func approxMatch(fst *pdf.PDF, snd *pdf.PDF, fstResolved map[string]bool, sndResolved map[string]bool, bestMatches map[string]string, bestScores map[string]float64, opts *pdf.MatchOptions) int {
	iteration := 0
	statApprox := 0
	for len(fstResolved) != len(fst.Objects) || len(sndResolved) != len(snd.Objects) {
		firstMatch := make(map[string]float64)
		secondMatch := make(map[string]float64)
		localMatches := make(map[string]string)

		for k1, o1 := range fst.Objects {

			// Skip perfect matched objects
			if fstResolved[k1] {
				continue
			}

			bestScore := 0.0
			bestKey := ""
			for k2, o2 := range snd.Objects {

				// Skip perfect matched objects
				if sndResolved[k2] {
					continue
				}

				// Calculate match
				score := pdf.MatchTypes(o1, o2, opts)
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
				if firstMatch[k1] < bestScore && secondMatch[bestKey] < bestScore {
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
				bestScores[k1] = firstMatch[k1]
				fstResolved[k1] = true
				sndResolved[k2] = true
				statApprox++
			}
		}

		iteration++
		if iteration > 100 {
			log.Fatalln("infinite loop detected")
		}
	}

	return statApprox
}

func diffPDF(firstPath string, secondPath string) {

	pdf.HideRandomKeys = true
	pdf.HideVariableData = true
	pdf.HideIdentifiers = true
	pdf.HideStreamLength = true
	pdf.TrimFontPrefix = true

	first := parsePDF(firstPath)
	second := parsePDF(secondPath)

	n1 := len(first.Objects)
	n2 := len(second.Objects)
	if n1 > n2 {
		n2, n1 = n1, n2
	}

	fmt.Printf("comparing %d with %d objects\n", len(first.Objects), len(second.Objects))

	bestMatches := make(map[string]string)
	bestMatchScores := make(map[string]float64)
	fstResolved := make(map[string]bool)
	sndResolved := make(map[string]bool)

	matches := 0
	for k1, o1 := range first.Objects {
		for k2, o2 := range second.Objects {

			// Skip perfect matched objects
			if sndResolved[k2] {
				continue
			}

			// Calculate match
			opts := pdf.MatchOptions{}
			score := pdf.MatchTypes(o1, o2, &opts)

			// Lock perfect matches
			if score == 1.0 {
				fstResolved[k1] = true
				sndResolved[k2] = true
				bestMatches[k1] = k2
				bestMatchScores[k1] = score
				matches++
				break
			}
		}
	}
	if matches > 0 {
		fmt.Printf("exact matches:\t%d\n", matches)
	}

	opts := pdf.MatchOptions{MatchDepth: true}
	matches = approxMatch(first, second, fstResolved, sndResolved, bestMatches, bestMatchScores, &opts)
	if matches > 0 {
		fmt.Printf("close matches:\t%d\n", matches)
	}

	opts.MatchDepth = false
	matches = approxMatch(first, second, fstResolved, sndResolved, bestMatches, bestMatchScores, &opts)
	if matches > 0 {
		fmt.Printf("distant matches:\t%d\n", matches)
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

	fstUnmatched := 0
	for k, v := range first.Objects {
		if !fstResolved[k] {
			_, _ = f1.WriteString(fmt.Sprintf("# Object Unmatched\n"))
			_, _ = f1.WriteString(v.String())
			fstUnmatched++
		}
	}

	sndUnmatched := 0
	for k, v := range second.Objects {
		if !sndResolved[k] {
			_, _ = f2.WriteString(fmt.Sprintf("# Object Unmatched\n"))
			_, _ = f2.WriteString(v.String())
			sndUnmatched++
		}
	}

	success := 1.0 - (math.Max(float64(fstUnmatched), float64(sndUnmatched))-float64(n2-n1))/float64(n1)
	fmt.Printf("match rate:\t%d%%\n", int(math.Round(success*100)))

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
