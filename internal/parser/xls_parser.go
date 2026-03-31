package parser

import (
	"fmt"
	"jarwise-backend/internal/models"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type XlsParser struct{}

type xlsHeader struct {
	date     int
	amount   int
	txType   int
	note     int
	account  int
	category int
}

func NewXlsParser() *XlsParser {
	return &XlsParser{}
}

// Parse reads the HTML (fake XLS) file and extracts transaction data.
func (p *XlsParser) Parse(filePath string) (*models.ParsedData, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	doc, err := html.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	rows := collectHTMLTableRows(doc)
	header, hasStructuredHeader := detectStructuredHeader(rows)

	result := &models.ParsedData{
		Transactions: []models.TransactionDTO{},
	}

	for _, row := range rows {
		if hasStructuredHeader && isHeaderRow(row, header) {
			continue
		}

		tx, ok := p.processRow(row, header, hasStructuredHeader)
		if !ok {
			continue
		}

		result.Transactions = append(result.Transactions, tx)
		if tx.Type == 1 {
			result.TotalIncome += tx.Amount
		} else if tx.Type == 0 {
			result.TotalExpense += tx.Amount
		}
	}

	return result, nil
}

func collectHTMLTableRows(root *html.Node) [][]string {
	var rows [][]string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var cols []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
					cols = append(cols, strings.TrimSpace(extractText(c)))
				}
			}
			if len(cols) > 0 {
				rows = append(rows, cols)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)

	return rows
}

func detectStructuredHeader(rows [][]string) (xlsHeader, bool) {
	for _, row := range rows {
		header, ok := buildHeader(row)
		if ok {
			return header, true
		}
	}
	return xlsHeader{}, false
}

func buildHeader(cols []string) (xlsHeader, bool) {
	header := xlsHeader{
		date:     -1,
		amount:   -1,
		txType:   -1,
		note:     -1,
		account:  -1,
		category: -1,
	}

	for index, col := range cols {
		switch normalizeXlsHeader(col) {
		case "date":
			header.date = index
		case "amount", "thb":
			if header.amount == -1 {
				header.amount = index
			}
		case "incomeexpense":
			header.txType = index
		case "note", "description":
			if header.note == -1 {
				header.note = index
			}
		case "account":
			if header.account == -1 {
				header.account = index
			}
		case "category":
			header.category = index
		}
	}

	return header, header.date != -1 && header.amount != -1 && header.txType != -1
}

func normalizeXlsHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "", "/", "", "-", "", "_", "")
	return replacer.Replace(value)
}

func isHeaderRow(cols []string, header xlsHeader) bool {
	candidate, ok := buildHeader(cols)
	if !ok {
		return false
	}
	return candidate.date == header.date && candidate.amount == header.amount && candidate.txType == header.txType
}

func (p *XlsParser) processRow(cols []string, header xlsHeader, hasStructuredHeader bool) (models.TransactionDTO, bool) {
	if hasStructuredHeader {
		return p.processStructuredRow(cols, header)
	}
	return p.processLegacyRow(cols)
}

func (p *XlsParser) processStructuredRow(cols []string, header xlsHeader) (models.TransactionDTO, bool) {
	date := valueAt(cols, header.date)
	if date == "" {
		return models.TransactionDTO{}, false
	}

	amount, ok := parseMoney(valueAt(cols, header.amount))
	if !ok {
		return models.TransactionDTO{}, false
	}

	txType, ok := parseXlsTransactionType(valueAt(cols, header.txType), amount)
	if !ok {
		return models.TransactionDTO{}, false
	}

	note := valueAt(cols, header.note)
	if note == "" {
		note = valueAt(cols, header.category)
	}

	return models.TransactionDTO{
		Date:       date,
		Amount:     math.Abs(amount),
		Type:       txType,
		AccountID:  valueAt(cols, header.account),
		CategoryID: valueAt(cols, header.category),
		Note:       note,
	}, true
}

func (p *XlsParser) processLegacyRow(cols []string) (models.TransactionDTO, bool) {
	if len(cols) < 5 {
		return models.TransactionDTO{}, false
	}

	amountIndex := -1
	var amount float64
	for index := len(cols) - 1; index >= 0; index-- {
		parsed, ok := parseMoney(cols[index])
		if ok {
			amountIndex = index
			amount = parsed
			break
		}
	}
	if amountIndex == -1 {
		return models.TransactionDTO{}, false
	}

	txType, ok := parseXlsTransactionType("", amount)
	if !ok {
		return models.TransactionDTO{}, false
	}

	note := ""
	if len(cols) > 3 {
		note = cols[3]
	}

	return models.TransactionDTO{
		Date:       valueAt(cols, 0),
		Amount:     math.Abs(amount),
		Type:       txType,
		AccountID:  valueAt(cols, 1),
		CategoryID: valueAt(cols, 2),
		Note:       note,
	}, true
}

func parseMoney(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "(", "-")
	value = strings.ReplaceAll(value, ")", "")

	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}

	return amount, true
}

func parseXlsTransactionType(kind string, amount float64) (int, bool) {
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	switch {
	case strings.Contains(normalizedKind, "income"):
		return 1, true
	case strings.Contains(normalizedKind, "expense"):
		return 0, true
	case strings.Contains(normalizedKind, "transfer"):
		return 2, true
	case amount > 0:
		return 1, true
	case amount < 0:
		return 0, true
	default:
		return 0, false
	}
}

func valueAt(cols []string, index int) string {
	if index < 0 || index >= len(cols) {
		return ""
	}
	return strings.TrimSpace(cols[index])
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += extractText(c)
	}
	return text
}
