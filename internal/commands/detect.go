package commands

import (
	"regexp"
	"strings"
)

var normalizePattern = regexp.MustCompile(`[^a-z0-9_:+]+`)

func DetectCommandRequest(question string) []CommandEntry {
	cleaned := strings.ToLower(question)
	normalized := normalizePattern.ReplaceAllString(cleaned, " ")
	tokens := strings.Fields(normalized)

	matches := make([]CommandEntry, 0)
	seen := make(map[string]struct{})

	for _, entry := range catalog {
		if detectMatch(cleaned, tokens, entry) {
			if _, ok := seen[entry.Code]; !ok {
				matches = append(matches, entry)
				seen[entry.Code] = struct{}{}
			}
		}
	}

	return matches
}

func detectMatch(cleaned string, tokens []string, entry CommandEntry) bool {
	for _, phrase := range entry.Triggers {
		if strings.Contains(cleaned, phrase) {
			return true
		}
	}

	for _, set := range entry.KeywordSets {
		if allKeywordsPresent(tokens, set) {
			return true
		}
	}

	if entry.AllowLabelMatch {
		label := strings.ToLower(entry.Label)
		code := strings.ToLower(entry.Code)
		if strings.Contains(cleaned, label) || strings.Contains(cleaned, code) {
			return true
		}
	}

	return false
}

func allKeywordsPresent(tokens []string, set []string) bool {
	if len(set) == 0 {
		return false
	}

	for _, keyword := range set {
		if !contains(tokens, keyword) {
			return false
		}
	}
	return true
}

func contains(tokens []string, keyword string) bool {
	for _, token := range tokens {
		if token == keyword {
			return true
		}
	}
	return false
}

func FormatCommandResponse(entry CommandEntry) string {
	extra := ""
	if strings.HasSuffix(entry.Code, ":") {
		extra = " Provide the required argument directly after the colon."
	}
	return "Use the `" + entry.Label + "` command (`" + entry.Code + "`) to " + entry.Description + extra
}
