// backend/internal/seeder/processor.go
package seeder

import (
	"regexp"
	"strings"
	"unicode"
)

// ContentProcessor handles text processing and cleanup
type ContentProcessor struct {
	// Regex patterns for cleaning content
	multiWhitespace *regexp.Regexp
	htmlTags        *regexp.Regexp
	wikiLinks       *regexp.Regexp
	codeBlocks      *regexp.Regexp
}

func NewContentProcessor() *ContentProcessor {
	return &ContentProcessor{
		multiWhitespace: regexp.MustCompile(`\s+`),
		htmlTags:        regexp.MustCompile(`<[^>]*>`),
		wikiLinks:       regexp.MustCompile(`\[\[[^\]]*\]\]`),
		codeBlocks:      regexp.MustCompile(`(?s)<code[^>]*>.*?</code>`),
	}
}

// CleanContent removes unwanted formatting and normalizes text
func (cp *ContentProcessor) CleanContent(content string) string {
	// Remove HTML tags
	content = cp.htmlTags.ReplaceAllString(content, "")
	
	// Remove wiki links but keep the text
	content = cp.wikiLinks.ReplaceAllStringFunc(content, func(link string) string {
		// Extract display text from [[Page|Display Text]] or [[Page]]
		link = strings.Trim(link, "[]")
		parts := strings.Split(link, "|")
		if len(parts) > 1 {
			return parts[1] // Return display text
		}
		return parts[0] // Return page name
	})
	
	// Normalize whitespace
	content = cp.multiWhitespace.ReplaceAllString(content, " ")
	
	// Remove excessive newlines
	lines := strings.Split(content, "\n")
	var cleaned []string
	emptyLines := 0
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			emptyLines++
			if emptyLines <= 2 { // Allow max 2 consecutive empty lines
				cleaned = append(cleaned, "")
			}
		} else {
			emptyLines = 0
			cleaned = append(cleaned, line)
		}
	}
	
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// ExtractCommandExamples finds command-line examples in content
func (cp *ContentProcessor) ExtractCommandExamples(content string) []string {
	var commands []string
	commandPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^\s*\$\s+([^\n]+)`),      // $ command
		regexp.MustCompile(`(?m)^\s*#\s+([^\n]+)`),       // # command
		regexp.MustCompile(`(?m)^\s*sudo\s+([^\n]+)`),    // sudo command
		regexp.MustCompile(`(?m)^\s*pacman\s+([^\n]+)`),  // pacman command
		regexp.MustCompile(`(?m)^\s*systemctl\s+([^\n]+)`), // systemctl command
	}
	
	for _, pattern := range commandPatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) > 1 {
				cmd := strings.TrimSpace(match[1])
				if len(cmd) > 3 && len(cmd) < 200 {
					commands = append(commands, cmd)
				}
			}
		}
	}
	
	return cp.removeDuplicates(commands)
}

// ExtractFilePaths finds file paths and configuration references
func (cp *ContentProcessor) ExtractFilePaths(content string) []string {
	var paths []string
	pathPatterns := []*regexp.Regexp{
		regexp.MustCompile(`/[a-zA-Z0-9\-_/\.]+\.conf`),
		regexp.MustCompile(`/[a-zA-Z0-9\-_/\.]+\.service`),
		regexp.MustCompile(`/etc/[a-zA-Z0-9\-_/\.]+`),
		regexp.MustCompile(`/usr/[a-zA-Z0-9\-_/\.]+`),
		regexp.MustCompile(`/var/[a-zA-Z0-9\-_/\.]+`),
		regexp.MustCompile(`/home/[a-zA-Z0-9\-_/\.]+`),
		regexp.MustCompile(`~/[a-zA-Z0-9\-_/\.]+`),
	}
	
	for _, pattern := range pathPatterns {
		matches := pattern.FindAllString(content, -1)
		for _, match := range matches {
			if len(match) > 3 && len(match) < 100 {
				paths = append(paths, match)
			}
		}
	}
	
	return cp.removeDuplicates(paths)
}

// ExtractErrorKeywords finds error-related keywords and phrases
func (cp *ContentProcessor) ExtractErrorKeywords(content string) []string {
	var keywords []string
	
	// Common error keywords in Arch Linux
	errorKeywords := []string{
		"error", "failed", "failure", "problem", "issue", "trouble",
		"cannot", "can't", "unable", "not working", "broken",
		"denied", "refused", "rejected", "forbidden",
		"missing", "not found", "no such", "does not exist",
		"timeout", "connection", "network", "unreachable",
		"permission", "access", "unauthorized", "forbidden",
		"conflict", "dependency", "package", "version",
		"kernel panic", "segmentation fault", "core dump",
		"service failed", "unit failed", "mount failed",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, keyword := range errorKeywords {
		if strings.Contains(contentLower, keyword) {
			keywords = append(keywords, keyword)
		}
	}
	
	return keywords
}

// SplitIntoChunks splits content into smaller chunks for better search
func (cp *ContentProcessor) SplitIntoChunks(content string, maxChunkSize int) []string {
	if len(content) <= maxChunkSize {
		return []string{content}
	}
	
	// Split by paragraphs first
	paragraphs := strings.Split(content, "\n\n")
	var chunks []string
	var currentChunk strings.Builder
	
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		
		// If adding this paragraph would exceed the limit, start a new chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+len(paragraph)+2 > maxChunkSize {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}
		
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(paragraph)
	}
	
	// Add the last chunk if it has content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	// If a single paragraph is too long, split it by sentences
	var finalChunks []string
	for _, chunk := range chunks {
		if len(chunk) <= maxChunkSize {
			finalChunks = append(finalChunks, chunk)
		} else {
			finalChunks = append(finalChunks, cp.splitBySentences(chunk, maxChunkSize)...)
		}
	}
	
	return finalChunks
}

// splitBySentences splits text by sentences when paragraphs are too long
func (cp *ContentProcessor) splitBySentences(text string, maxSize int) []string {
	sentences := regexp.MustCompile(`[.!?]+\s+`).Split(text, -1)
	var chunks []string
	var currentChunk strings.Builder
	
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		
		if currentChunk.Len() > 0 && currentChunk.Len()+len(sentence)+2 > maxSize {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}
		
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(". ")
		}
		currentChunk.WriteString(sentence)
	}
	
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}
	
	return chunks
}

// removeDuplicates removes duplicate strings from a slice
func (cp *ContentProcessor) removeDuplicates(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// CountWords estimates word count in text
func (cp *ContentProcessor) CountWords(text string) int {
	if text == "" {
		return 0
	}
	
	// Split by whitespace and count
	words := strings.FieldsFunc(text, func(c rune) bool {
		return unicode.IsSpace(c) || unicode.IsPunct(c)
	})
	
	// Filter out very short "words"
	count := 0
	for _, word := range words {
		if len(strings.TrimSpace(word)) > 1 {
			count++
		}
	}
	
	return count
}

// CalculateReadability gives a basic readability score (0-100)
func (cp *ContentProcessor) CalculateReadability(text string) int {
	if text == "" {
		return 0
	}
	
	wordCount := cp.CountWords(text)
	sentenceCount := len(regexp.MustCompile(`[.!?]+`).Split(text, -1))
	
	if sentenceCount == 0 {
		return 50 // Default middle score
	}
	
	avgWordsPerSentence := float64(wordCount) / float64(sentenceCount)
	
	// Simple readability calculation
	// Lower score = easier to read, higher score = harder
	// We invert this to make higher scores better
	score := 100 - int(avgWordsPerSentence*2)
	
	// Clamp to 0-100 range
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	
	return score
}

// ExtractMetaTags extracts metadata from content
func (cp *ContentProcessor) ExtractMetaTags(content string) map[string]string {
	meta := make(map[string]string)
	
	// Extract category information
	if strings.Contains(strings.ToLower(content), "troubleshoot") {
		meta["category"] = "troubleshooting"
	} else if strings.Contains(strings.ToLower(content), "install") {
		meta["category"] = "installation"
	} else if strings.Contains(strings.ToLower(content), "config") {
		meta["category"] = "configuration"
	} else {
		meta["category"] = "general"
	}
	
	// Extract difficulty level
	commandCount := len(cp.ExtractCommandExamples(content))
	if commandCount > 10 {
		meta["difficulty"] = "advanced"
	} else if commandCount > 3 {
		meta["difficulty"] = "intermediate"
	} else {
		meta["difficulty"] = "beginner"
	}
	
	// Extract topic
	contentLower := strings.ToLower(content)
	topics := []string{"pacman", "systemd", "grub", "xorg", "wayland", "network", "audio", "video", "kernel"}
	for _, topic := range topics {
		if strings.Contains(contentLower, topic) {
			meta["topic"] = topic
			break
		}
	}
	
	return meta
}