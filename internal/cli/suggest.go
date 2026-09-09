package cli

import "strings"

// Levenshtein calculates the Levenshtein distance between two strings.
func Levenshtein(a, b string) int {
	la := len(a)
	lb := len(b)
	d := make([][]int, la+1)
	for i := range d {
		d[i] = make([]int, lb+1)
		d[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		d[0][j] = j
	}

	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = min(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)
		}
	}
	return d[la][lb]
}

func min(vals ...int) int {
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// Suggest finds the closest matching candidate from a list of candidates.
// Returns empty string if no candidate is sufficiently close.
func Suggest(target string, candidates []string) string {
	bestCandidate := ""
	bestDist := 999

	targetLower := strings.ToLower(target)

	for _, cand := range candidates {
		candLower := strings.ToLower(cand)
		dist := Levenshtein(targetLower, candLower)

		// Heuristic: distance must be reasonably small relative to word length
		maxAllowed := 3
		if len(target) <= 4 {
			maxAllowed = 2
		}

		if dist <= maxAllowed && dist < bestDist {
			bestDist = dist
			bestCandidate = cand
		}
	}

	return bestCandidate
}
