package vault

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

var (
	// ErrNoClosingDelimiter indicates an opening --- was found but no closing ---.
	ErrNoClosingDelimiter = errors.New("opening frontmatter delimiter found but no closing delimiter")
)

var delimiter = []byte("---")

// Parse extracts YAML frontmatter from markdown content.
// Returns the frontmatter bytes (without delimiters), the body bytes, and any error.
// If the content has no frontmatter (first line is not ---), returns nil frontmatter
// and the full content as body with no error.
func Parse(content []byte) (frontmatter []byte, body []byte, err error) {
	lines := splitLines(content)
	if len(lines) == 0 {
		return nil, content, nil
	}

	// Opening delimiter must be the first line.
	if !isDelimiter(lines[0]) {
		return nil, content, nil
	}

	// Find the closing delimiter.
	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if isDelimiter(lines[i]) {
			closingIdx = i
			break
		}
	}

	if closingIdx < 0 {
		return nil, nil, ErrNoClosingDelimiter
	}

	// Extract frontmatter (between delimiters, may be empty).
	if closingIdx == 1 {
		frontmatter = []byte{}
	} else {
		frontmatter = joinLines(lines[1:closingIdx])
	}

	// Extract body (everything after closing delimiter).
	if closingIdx+1 < len(lines) {
		body = joinLines(lines[closingIdx+1:])
	}

	return frontmatter, body, nil
}

// Strip removes YAML frontmatter from content, returning only the body.
// If parsing fails or there is no frontmatter, returns the original content.
func Strip(content []byte) []byte {
	_, body, err := Parse(content)
	if err != nil || body == nil {
		return content
	}
	return body
}

// Generate creates a YAML frontmatter block from a struct value.
// The result includes the opening and closing --- delimiters.
func Generate(v any) ([]byte, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal frontmatter: %w", err)
	}

	var buf bytes.Buffer
	buf.Write(delimiter)
	buf.WriteByte('\n')
	buf.Write(data)
	buf.Write(delimiter)
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// isDelimiter checks if a line (after trimming \r) equals "---".
func isDelimiter(line []byte) bool {
	return bytes.Equal(bytes.TrimRight(line, "\r"), delimiter)
}

// splitLines splits content into lines, preserving line content without the newline.
func splitLines(content []byte) [][]byte {
	if len(content) == 0 {
		return nil
	}
	lines := bytes.Split(content, []byte("\n"))
	// bytes.Split on trailing \n produces an empty final element — remove it.
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// joinLines rejoins lines with newline separators and a trailing newline.
func joinLines(lines [][]byte) []byte {
	if len(lines) == 0 {
		return nil
	}
	result := bytes.Join(lines, []byte("\n"))
	result = append(result, '\n')
	return result
}
