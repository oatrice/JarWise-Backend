package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestXlsParser_ParseStructuredMoneyManagerExport(t *testing.T) {
	parser := NewXlsParser()
	path := writeTempXlsFixture(t, `<html><body><table>
<tr><th>Date</th><th>Account</th><th>Category</th><th>Subcategory</th><th>Note</th><th>THB</th><th>Income/Expense</th><th>Description</th><th>Amount</th><th>Currency</th><th>Account</th></tr>
<tr><td>01/22/2025 08:15:00</td><td>Cash Wallet</td><td>Food</td><td></td><td>Lunch</td><td>100.50</td><td>Expense</td><td></td><td>100.50</td><td>THB</td><td>100.50</td></tr>
<tr><td>01/20/2025 09:00:00</td><td>Bank Account</td><td>Salary</td><td></td><td>Monthly Salary</td><td>50000.00</td><td>Income</td><td></td><td>50000.00</td><td>THB</td><td>50000.00</td></tr>
<tr><td>01/25/2025 12:00:00</td><td>Cash Wallet</td><td>Bank Account</td><td></td><td>Transfer to savings</td><td>5000.00</td><td>Transfer-Out</td><td></td><td>5000.00</td><td>THB</td><td>5000.00</td></tr>
</table></body></html>`)

	result, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Transactions) != 3 {
		t.Fatalf("expected 3 transactions, got %d", len(result.Transactions))
	}
	if result.TotalIncome != 50000 {
		t.Fatalf("expected total income 50000, got %.2f", result.TotalIncome)
	}
	if result.TotalExpense != 100.50 {
		t.Fatalf("expected total expense 100.50, got %.2f", result.TotalExpense)
	}
	if result.Transactions[2].Type != 2 {
		t.Fatalf("expected transfer row to parse as type 2, got %d", result.Transactions[2].Type)
	}
}

func TestXlsParser_ParseLegacySignedRows(t *testing.T) {
	parser := NewXlsParser()
	path := writeTempXlsFixture(t, `<html><body><table>
<tr><td>2025-01-15</td><td>Cash Wallet</td><td>Food</td><td>Lunch</td><td>-100.50</td></tr>
<tr><td>2025-01-20</td><td>Bank Account</td><td>Salary</td><td>Monthly Salary</td><td>50000.00</td></tr>
</table></body></html>`)

	result, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.Transactions) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(result.Transactions))
	}
	if result.TotalIncome != 50000 {
		t.Fatalf("expected total income 50000, got %.2f", result.TotalIncome)
	}
	if result.TotalExpense != 100.50 {
		t.Fatalf("expected total expense 100.50, got %.2f", result.TotalExpense)
	}
}

func writeTempXlsFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "fixture.xls")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write xls fixture: %v", err)
	}
	return path
}
