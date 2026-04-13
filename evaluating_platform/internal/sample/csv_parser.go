package sample

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// CsvRow represents a single row from a two-column CSV (index + data, no header).
type CsvRow struct {
	Index int
	Data  string
}

// ParseTwoColumnCSV parses CSV data in two-column format (integer index, data string).
// The CSV must have no header row. Each row must have exactly 2 columns,
// with the first column being a valid integer.
// Handles UTF-8 BOM that Excel may prepend to CSV files.
func ParseTwoColumnCSV(data []byte) ([]CsvRow, error) {
	normalized, err := NormalizeCSVData(data)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(bytes.NewReader(normalized))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	rows := make([]CsvRow, 0, len(records))
	for i, record := range records {
		if len(record) != 2 {
			return nil, fmt.Errorf("CSV must have exactly 2 columns: index (integer) and data")
		}

		idx, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			return nil, fmt.Errorf("column 1 must be integer, got '%s' at row %d", record[0], i+1)
		}

		rows = append(rows, CsvRow{
			Index: idx,
			Data:  record[1],
		})
	}

	return rows, nil
}

// NormalizeCSVData converts common spreadsheet encodings to UTF-8 before parsing.
func NormalizeCSVData(data []byte) ([]byte, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}), nil
	}
	if utf8.Valid(data) {
		return data, nil
	}

	if bytes.HasPrefix(data, []byte{0xFF, 0xFE}) || bytes.HasPrefix(data, []byte{0xFE, 0xFF}) {
		decoded, err := decodeText(data, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())
		if err == nil && utf8.Valid(decoded) {
			return decoded, nil
		}
	}

	if looksLikeUTF16(data) {
		for _, decoder := range []transform.Transformer{
			unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder(),
			unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder(),
		} {
			decoded, err := decodeText(data, decoder)
			if err == nil && utf8.Valid(decoded) {
				return decoded, nil
			}
		}
	}

	for _, decoder := range []transform.Transformer{
		simplifiedchinese.GB18030.NewDecoder(),
		simplifiedchinese.GBK.NewDecoder(),
	} {
		decoded, err := decodeText(data, decoder)
		if err == nil && utf8.Valid(decoded) {
			return decoded, nil
		}
	}

	return nil, fmt.Errorf("unsupported CSV encoding, please save as UTF-8/GBK/UTF-16 and retry")
}

func decodeText(data []byte, decoder transform.Transformer) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), decoder)
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return bytes.TrimPrefix(decoded, []byte{0xEF, 0xBB, 0xBF}), nil
}

func looksLikeUTF16(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	zeroCount := 0
	sample := data
	if len(sample) > 128 {
		sample = sample[:128]
	}
	for _, b := range sample {
		if b == 0 {
			zeroCount++
		}
	}
	return zeroCount >= len(sample)/6
}
