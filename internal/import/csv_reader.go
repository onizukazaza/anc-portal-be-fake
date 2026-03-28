package importer

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
)

type CSVData struct {
	Header []string
	Rows   [][]string
}

func ReadCSV(filePath string) (*CSVData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open csv file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv file: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("csv file is empty")
	}

	header := normalizeHeader(records[0])

	return &CSVData{
		Header: header,
		Rows:   records[1:],
	}, nil
}

func normalizeHeader(header []string) []string {
	result := make([]string, 0, len(header))
	for _, h := range header {
		result = append(result, strings.ToLower(strings.TrimSpace(h)))
	}
	return result
}

func headerIndexMap(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[h] = i
	}
	return m
}

func getCell(row []string, indexMap map[string]int, key string) string {
	idx, ok := indexMap[key]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func isEmptyRow(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}
