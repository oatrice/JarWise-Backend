# Luma Code Review Report

**Date:** 2026-03-10 11:20:04
**Files Reviewed:** ['internal/models/errors.go', 'code_review.md', 'internal/models/domain.go', 'internal/validator/validator.go', 'internal/repository/wallet_repository_test.go', 'internal/service/graph_service.go', '.gitignore', 'go.mod', 'internal/api/handlers/chart_handler.go', 'internal/api/handlers/graph_handler.go', 'internal/api/router.go', 'internal/models/chart.go', 'internal/importer/migration_integrity_test.go', 'internal/api/handlers/wallet_handler.go', 'internal/service/chart_service.go', 'internal/repository/wallet_repository.go', 'internal/importer/importer.go', 'internal/service/chart_service_test.go', 'internal/db/sqlite.go', 'internal/repository/transaction_repository.go', 'internal/service/transaction_service.go', 'cmd/server/main.go', 'internal/models/graph.go', 'internal/service/sync_conflict_test.go']

## 📝 Reviewer Feedback

สวัสดีครับ โค้ดที่ส่งมามีข้อผิดพลาดร้ายแรง 2 จุดครับ

1.  **JSON struct tags ไม่ถูกต้อง:** ตามที่ระบุไว้ในไฟล์ `code_review.md` โค้ดในไฟล์ `internal/models/graph.go`, `internal/models/chart.go`, และ `internal/api/handlers/graph_handler.go` ยังคงมี JSON struct tag ที่ผิดรูปแบบ คือมีการใช้ backslash (`\`) เกินเข้ามา ทำให้ `encoding/json` ไม่สามารถแปลงชื่อฟิลด์ได้อย่างถูกต้อง

    **ตัวอย่างที่ผิด:**
    ```go
    `json:\"label\"`
    ```

    **ที่ถูกต้องคือ:**
    ```go
    `json:"label"`
    ```
    ต้องลบ `\` ออกจาก struct tags ทั้งหมดในไฟล์ที่กล่าวมาครับ

2.  **การสร้าง SQL Query มีบั๊ก:** ในไฟล์ `internal/repository/transaction_repository.go` ฟังก์ชัน `GetExpenseGraphData` มีการใช้ `fmt.Sprintf` เพื่อสร้าง SQL query อย่างไม่ถูกต้อง โดยตัวแปร `dateFormat` (เช่น `"%Y-%m"`) มีอักขระ `%` ซึ่ง `fmt.Sprintf` จะพยายามตีความเป็น format specifier ทำให้ query ที่ได้ผิดพลาดและโปรแกรมจะทำงานไม่สำเร็จ

    **วิธีการแก้ไข** คือเปลี่ยนจากการใช้ `fmt.Sprintf` เป็นการต่อสตริง (string concatenation) แทน เพื่อป้องกันการตีความอักขระ `%` ที่ผิดพลาด

    **โค้ดที่มีปัญหา:**
    ```go
    query := fmt.Sprintf(`
        SELECT 
            strftime('%s', date) as period_label, 
            ...
    `, dateFormat)
    ```

    **โค้ดที่แก้ไขแล้ว:**
    ```go
    query := `
        SELECT 
            strftime('` + dateFormat + `', date) as period_label, 
            ...
    `
    ```

## 🧪 Test Suggestions

สวัสดีครับ เพื่อที่จะเขียน "Manual Verification Guide" ได้ ผมต้องการเห็นรายละเอียดของการเปลี่ยนแปลงโค้ดที่คุณว่าครับ

กรุณาส่งโค้ดที่เปลี่ยนแปลงมาให้ผมดูหน่อยครับ แล้วผมจะเขียนขั้นตอนการตรวจสอบแบบ step-by-step พร้อมผลลัพธ์ที่คาดหวังให้ครับ

