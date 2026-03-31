# Implementation Plan - แก้ไข Go Formatting และ TypeScript Lint Error

แก้ไขปัญหา Go code ไม่ได้ฟอร์แมตตามมาตรฐาน และแก้ไข Lint error ในไฟล์ React component ที่มีการใช้ `any` type

## User Review Required

> [!IMPORTANT]
> จะมีการแก้ไข Type ของ error ใน `ReportsPage.tsx` จาก `any` เป็น `unknown` และทำการ type checking ก่อนเข้าถึง property `message` เพื่อให้ผ่านกฎ ESLint `no-explicit-any`

## Proposed Changes

### Backend (Go Formatting)

#### [MODIFY] [Backend directory](file:///Users/oatrice/Software-projects/JarWise/Backend)
- รันคำสั่ง `go fmt ./...` เพื่อจัดรูปแบบโค้ดในไฟล์ต่อไปนี้:
    - `cmd/seed/main.go`
    - `cmd/seed-10-years/main.go`
    - `internal/api/cors_test.go`
    - `internal/service/report_service.go`
    - `internal/service/report_service_test.go`

### Web (TypeScript Lint)

#### [MODIFY] [ReportsPage.tsx](file:///Users/oatrice/Software-projects/JarWise/Web/src/pages/ReportsPage.tsx)
- แก้ไขบรรทัดที่ 80 จาก `catch (err: any)` เป็น `catch (err)` และปรับการเรียกใช้ `err.message` ให้ปลอดภัยยิ่งขึ้น

## Verification Plan

### Automated Tests
- **Backend**: รัน `go fmt -l ./...` เพื่อตรวจสอบว่าไม่มีไฟล์ไหนต้องฟอร์แมตเพิ่ม
- **Web**: รัน `npm run lint` ในโฟลเดอร์ `Web` เพื่อตรวจสอบว่า error หายไป

### Manual Verification
- ตรวจสอบความถูกต้องของโค้ดที่ถูกฟอร์แมตใหม่ด้วยตาเปล่า (Visual Check)
