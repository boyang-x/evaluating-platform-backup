package sample

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
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

// TemplateRow represents one uploaded jailbreak template row.
type TemplateRow struct {
	Index    int
	Category string
	Content  string
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

// ParseTemplateRows parses the current template upload format:
// index, category name, template content. It supports CSV and simple XLSX
// workbooks exported by spreadsheet tools. A header row is optional.
func ParseTemplateRows(data []byte, filename string) ([]TemplateRow, error) {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext == ".xlsx" {
		return parseTemplateRowsXLSX(data)
	}
	return parseTemplateRowsCSV(data)
}

func parseTemplateRowsCSV(data []byte) ([]TemplateRow, error) {
	normalized, err := NormalizeCSVData(data)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(bytes.NewReader(normalized))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse CSV: %w", err)
	}
	return parseTemplateRecords(records)
}

func parseTemplateRowsXLSX(data []byte) ([]TemplateRow, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("parse XLSX: %w", err)
	}
	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}
	sharedStrings, err := readXLSXSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return nil, err
	}
	sheet := files["xl/worksheets/sheet1.xml"]
	if sheet == nil {
		return nil, fmt.Errorf("XLSX must contain xl/worksheets/sheet1.xml")
	}
	records, err := readXLSXSheetRows(sheet, sharedStrings)
	if err != nil {
		return nil, err
	}
	return parseTemplateRecords(records)
}

func parseTemplateRecords(records [][]string) ([]TemplateRow, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("template file is empty")
	}
	rows := make([]TemplateRow, 0, len(records))
	for i, record := range records {
		if isEmptyRecord(record) {
			continue
		}
		if len(record) < 3 {
			return nil, fmt.Errorf("template file must have 3 columns: index, category name, template content")
		}
		idx, err := strconv.Atoi(strings.TrimSpace(record[0]))
		if err != nil {
			if i == 0 && looksLikeTemplateHeader(record) {
				continue
			}
			return nil, fmt.Errorf("column 1 must be integer, got '%s' at row %d", record[0], i+1)
		}
		category := strings.TrimSpace(record[1])
		content := strings.TrimSpace(record[2])
		if category == "" {
			return nil, fmt.Errorf("column 2 category name is required at row %d", i+1)
		}
		if content == "" {
			return nil, fmt.Errorf("column 3 template content is required at row %d", i+1)
		}
		rows = append(rows, TemplateRow{Index: idx, Category: category, Content: content})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("template file has no data rows")
	}
	return rows, nil
}

func looksLikeTemplateHeader(record []string) bool {
	if len(record) < 3 {
		return false
	}
	joined := strings.ToLower(strings.Join(record[:3], " "))
	return strings.Contains(joined, "序号") || strings.Contains(joined, "分类") || strings.Contains(joined, "模板") ||
		strings.Contains(joined, "index") || strings.Contains(joined, "category") || strings.Contains(joined, "template")
}

func isEmptyRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

type xlsxSharedStringTable struct {
	Items []struct {
		Texts []string `xml:"t"`
	} `xml:"si"`
}

func readXLSXSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open shared strings: %w", err)
	}
	defer rc.Close()
	var table xlsxSharedStringTable
	if err := xml.NewDecoder(rc).Decode(&table); err != nil {
		return nil, fmt.Errorf("parse shared strings: %w", err)
	}
	out := make([]string, 0, len(table.Items))
	for _, item := range table.Items {
		out = append(out, strings.Join(item.Texts, ""))
	}
	return out, nil
}

type xlsxWorksheet struct {
	Rows []xlsxRow `xml:"sheetData>row"`
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref       string `xml:"r,attr"`
	Type      string `xml:"t,attr"`
	Value     string `xml:"v"`
	InlineStr struct {
		Text string `xml:"t"`
	} `xml:"is"`
}

func readXLSXSheetRows(file *zip.File, sharedStrings []string) ([][]string, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open worksheet: %w", err)
	}
	defer rc.Close()
	var sheet xlsxWorksheet
	if err := xml.NewDecoder(rc).Decode(&sheet); err != nil {
		return nil, fmt.Errorf("parse worksheet: %w", err)
	}
	records := make([][]string, 0, len(sheet.Rows))
	for _, row := range sheet.Rows {
		values := []string{}
		for i, cell := range row.Cells {
			col := xlsxColumnIndex(cell.Ref)
			if col < 0 {
				col = i
			}
			for len(values) <= col {
				values = append(values, "")
			}
			values[col] = xlsxCellValue(cell, sharedStrings)
		}
		records = append(records, values)
	}
	return records, nil
}

func xlsxCellValue(cell xlsxCell, sharedStrings []string) string {
	switch cell.Type {
	case "s":
		idx, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err == nil && idx >= 0 && idx < len(sharedStrings) {
			return sharedStrings[idx]
		}
	case "inlineStr":
		return cell.InlineStr.Text
	}
	return cell.Value
}

func xlsxColumnIndex(ref string) int {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return -1
	}
	col := 0
	seen := false
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			col = col*26 + int(r-'A'+1)
			seen = true
			continue
		}
		if r >= 'a' && r <= 'z' {
			col = col*26 + int(r-'a'+1)
			seen = true
			continue
		}
		break
	}
	if !seen {
		return -1
	}
	return col - 1
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
