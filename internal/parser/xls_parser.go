package parser

import (
	"fmt"
	"jarwise-backend/internal/models"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type XlsParser struct{}

func NewXlsParser() *XlsParser {
	return &XlsParser{}
}

// Parse reads the HTML (fake XLS) file and extracts transaction data
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

	result := &models.ParsedData{
		Transactions: []models.TransactionDTO{},
	}

	// Traverse HTML to find rows
	var extractRows func(*html.Node)
	extractRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			// Found a row, process cells
			if t, ok := p.processRow(n); ok {
				result.Transactions = append(result.Transactions, t)
				if t.Type == 1 {
					result.TotalIncome += t.Amount
				} else if t.Type == 0 {
					result.TotalExpense += t.Amount
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractRows(c)
		}
	}
	extractRows(doc)

	return result, nil
}

func (p *XlsParser) processRow(tr *html.Node) (models.TransactionDTO, bool) {
	var cols []string
	for c := tr.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td") {
			text := extractText(c)
			cols = append(cols, strings.TrimSpace(text))
		}
	}

	// Basic validation for Money Manager format
	// Usually: Date, Account, Category, Note, Amount, etc.
	// We need to know the *exact* column order.
	// For MVP, checking if it looks like a data row (has enough columns and a date)
	if len(cols) < 5 {
		return models.TransactionDTO{}, false
	}

	// Attempt to parse amount (assuming it's in a specific column, e.g., last or near last)
	// This is fragile without the exact headers.
	// Assuming column index 0 is Date, and finding Amount...
	// Let's assume standard export: Date, Asset, Category, Content, Amount, (Type inferred)

	// Example heuristic:
	// Col 0: Date (YYYY-MM-DD)
	// Col 4: Amount (+/-)

	amountStr := strings.ReplaceAll(cols[len(cols)-1], ",", "") // Remove commas
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return models.TransactionDTO{}, false // Title row or invalid
	}

	t := models.TransactionDTO{
		Date:   cols[0],
		Amount: amount, // Logic to split income/expense needed if signed
		// Note: Money Manager usage usually:
		// Income is positive, Expense is negative in XLS? Or separated?
		// We'll assume signed for now.
	}

	if amount > 0 {
		t.Type = 1 // Income
	} else {
		t.Type = 0         // Expense
		t.Amount = -amount // Store absolute in DB usually? checking DTO..
		// Our DTO usually wants positive amount + Type.
		// Let's normalize.
	}

	return t, true
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
