# Issue #50: Robust Offline and Server Error Handling

## Planning
- [x] อ่านเอกสาร feature ทั้งหมด
- [x] สำรวจ codebase ทั้ง 3 platforms
- [x] เขียน implementation plan

## Implementation (Error Handling)
- [x] Web: ErrorState component + integration
- [x] Android: ErrorState + ViewModel + Repository
- [x] iOS: ErrorStateView + ViewModel + Repository

## Code Review Fixes

### Race Condition (Critical) — ทั้ง 3 Platforms
- [/] Web: แก้ `setWeeklyData(originalData)` → revert เฉพาะ item
- [ ] Android: แก้ `fetchWeek()` ใน catch → revert เฉพาะ item
- [ ] iOS: แก้ `loadWeeklyData(forceRefresh)` → revert เฉพาะ item

### Architectural Issues
- [ ] Android: แยก `refreshWeek()` ออกจาก `onStart`
- [ ] Android: แปลง AuthStateListener เป็น callbackFlow
- [ ] iOS: ย้าย @StateObject ไปที่ Parent View
- [ ] iOS: สร้าง WisdomGardenError enum

### Backend
- [ ] เพิ่ม `FindByID` ใน category repository
- [ ] แก้ `DeleteCategory` ให้ใช้ `FindByID`
- [ ] Test: DeleteCategory UsageCount==0
- [ ] Test: Reorder IDs ไม่ตรง
- [ ] Test: FindAll JOIN fail fallback

## Verification
- [ ] รัน unit tests ทุก platform
