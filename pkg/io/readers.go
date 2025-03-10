package io

import (
	"bufio"
	"io"
)

func ReadLines(reader io.Reader, sanitizers ...func(string) string) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var lines []string
	for scanner.Scan() {
		l := scanner.Text()
		for _, sanitizer := range sanitizers {
			l = sanitizer(l)
		}
		if l == "" {
			continue
		}
		lines = append(lines, l)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
