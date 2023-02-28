package pdfdiff

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	pdf2 "pdfdump/external/pdf"
	"pdfdump/internal/token"
)

func parsePDF(filePath string) *pdf2.PDF {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err)
	}
	scanner := token.NewScanner(f)
	parser := pdf2.NewParser(scanner)
	parser.Parse()
	_ = f.Close()
	return parser.PDF()
}

type Comparison struct {
	LeftPath    string
	RightPath   string
	LeftOutput  string
	RightOutput string
}

func approxMatch(left *pdf2.PDF, right *pdf2.PDF, leftResolved map[string]bool, rightResolved map[string]bool, bestMatches map[string]string, bestScores map[string]float64, opts *pdf2.MatchOptions) int {
	iteration := 0
	statApprox := 0
	for len(leftResolved) != len(left.Objects) || len(rightResolved) != len(right.Objects) {
		leftMatch := make(map[string]float64)
		rightMatch := make(map[string]float64)
		localMatches := make(map[string]string)

		for k1, o1 := range left.Objects {

			// Skip perfect matched objects
			if leftResolved[k1] {
				continue
			}

			bestScore := 0.0
			bestKey := ""
			for k2, o2 := range right.Objects {

				// Skip perfect matched objects
				if rightResolved[k2] {
					continue
				}

				// Calculate match
				score := pdf2.MatchTypes(o1, o2, opts)
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
				if leftMatch[k1] < bestScore && rightMatch[bestKey] < bestScore {
					localMatches[k1] = bestKey
				}
				leftMatch[k1] = math.Max(leftMatch[k1], bestScore)
				rightMatch[bestKey] = math.Max(rightMatch[bestKey], bestScore)
			}
		}

		if len(localMatches) == 0 {
			break
		}

		for k1, k2 := range localMatches {
			if leftMatch[k1] == rightMatch[k2] {
				bestMatches[k1] = k2
				bestScores[k1] = leftMatch[k1]
				leftResolved[k1] = true
				rightResolved[k2] = true
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

func Compare(leftPath string, rightPath string, verbose bool) *Comparison {

	pdf2.HideRandomKeys = true
	pdf2.HideVariableData = true
	pdf2.HideIdentifiers = true
	pdf2.HideStreamLength = true
	pdf2.TrimFontPrefix = true

	left := parsePDF(leftPath)
	right := parsePDF(rightPath)

	n1 := len(left.Objects)
	n2 := len(right.Objects)
	if n1 > n2 {
		n2, n1 = n1, n2
	}

	if verbose {
		fmt.Printf("comparing %d with %d objects\n", len(left.Objects), len(right.Objects))
	}

	bestMatches := make(map[string]string)
	bestMatchScores := make(map[string]float64)
	leftResolved := make(map[string]bool)
	rightResolved := make(map[string]bool)

	matches := 0
	for k1, o1 := range left.Objects {
		for k2, o2 := range right.Objects {

			// Skip perfect matched objects
			if rightResolved[k2] {
				continue
			}

			// Calculate match
			opts := pdf2.MatchOptions{}
			score := pdf2.MatchTypes(o1, o2, &opts)

			// Lock perfect matches
			if score == 1.0 {
				leftResolved[k1] = true
				rightResolved[k2] = true
				bestMatches[k1] = k2
				bestMatchScores[k1] = score
				matches++
				break
			}
		}
	}
	if matches > 0 && verbose {
		fmt.Printf("exact matches:\t%d\n", matches)
	}

	opts := pdf2.MatchOptions{MatchDepth: true}
	matches = approxMatch(left, right, leftResolved, rightResolved, bestMatches, bestMatchScores, &opts)
	if matches > 0 && verbose {
		fmt.Printf("close matches:\t%d\n", matches)
	}

	opts.MatchDepth = false
	matches = approxMatch(left, right, leftResolved, rightResolved, bestMatches, bestMatchScores, &opts)
	if matches > 0 && verbose {
		fmt.Printf("distant matches:\t%d\n", matches)
	}

	leftBuffer := bytes.Buffer{}
	rightBuffer := bytes.Buffer{}

	index := 0
	for k1, k2 := range bestMatches {
		score := int(math.Round(bestMatchScores[k1] * 100))
		_, _ = leftBuffer.WriteString(fmt.Sprintf("# Object (%d) (%d%%)\n", index, score))
		_, _ = rightBuffer.WriteString(fmt.Sprintf("# Object (%d) (%d%%)\n", index, score))
		_, _ = leftBuffer.WriteString(left.Objects[k1].String())
		_, _ = rightBuffer.WriteString(right.Objects[k2].String())
		index++
	}

	fstUnmatched := 0
	for k, v := range left.Objects {
		if !leftResolved[k] {
			_, _ = leftBuffer.WriteString(fmt.Sprintf("# Object Unmatched\n"))
			_, _ = leftBuffer.WriteString(v.String())
			fstUnmatched++
		}
	}

	sndUnmatched := 0
	for k, v := range right.Objects {
		if !rightResolved[k] {
			_, _ = rightBuffer.WriteString(fmt.Sprintf("# Object Unmatched\n"))
			_, _ = rightBuffer.WriteString(v.String())
			sndUnmatched++
		}
	}

	if verbose {
		success := 1.0 - (math.Max(float64(fstUnmatched), float64(sndUnmatched))-float64(n2-n1))/float64(n1)
		fmt.Printf("match rate:\t%d%%\n", int(math.Round(success*100)))
	}

	return &Comparison{
		LeftPath:    leftPath,
		RightPath:   rightPath,
		LeftOutput:  leftBuffer.String(),
		RightOutput: rightBuffer.String(),
	}
}
