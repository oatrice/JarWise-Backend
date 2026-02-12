package models

// ChartData เป็น response struct สำหรับ chart API ที่รวมข้อมูลทั้งหมดที่ frontend ต้องการ
type ChartData struct {
	Summary    ChartSummary     `json:"summary"`
	Trend      []TrendPoint     `json:"trend"`
	ByCategory []CategoryAmount `json:"by_category"`
	ByJar      []JarAmount      `json:"by_jar"`
	Comparison *ComparisonData  `json:"comparison,omitempty"`
}

// ChartSummary สรุปรายรับ-รายจ่ายรวม
type ChartSummary struct {
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Net     float64 `json:"net"`
}

// TrendPoint ข้อมูล 1 จุดบน Line chart (แต่ละช่วงเวลา)
type TrendPoint struct {
	Date    string  `json:"date"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

// CategoryAmount ข้อมูลรายจ่ายแยกตาม Category/Jar
type CategoryAmount struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// JarAmount ข้อมูลการกระจายตัวตาม Jar
type JarAmount struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

// ComparisonData ข้อมูลเปรียบเทียบ 2 ช่วงเวลา
type ComparisonData struct {
	Current  ChartSummary `json:"current"`
	Previous ChartSummary `json:"previous"`
}
