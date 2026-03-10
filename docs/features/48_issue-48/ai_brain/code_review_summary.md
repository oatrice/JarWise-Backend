# สรุป Code Review จาก Luma CLI — Issue #50

ไฟล์ `code_review.md` ทั้ง 5 ไฟล์ (Root, Android, iOS, Web, Backend) ถูกสร้างโดย Luma CLI วันที่ 2026-03-08

---

## 🔴 Critical Issues (ต้องแก้)

### 1. Race Condition ใน Optimistic Update → **ทั้ง 3 Client Platforms**

> [!CAUTION]
> ปัญหาเดียวกันในทุกแพลตฟอร์ม: เมื่อ Toggle item ล้มเหลว ระบบจะ Revert **ทั้งหมด** แทนที่จะ Revert เฉพาะ item ที่ล้มเหลว

| Platform | ปัญหา | ไฟล์ |
|----------|-------|------|
| **Web** | `setWeeklyData(originalData)` ← Revert ทั้งก้อน ถ้า User toggle A แล้ว toggle B อย่างรวดเร็ว เมื่อ A fail จะ revert B ไปด้วย | [useWisdomGarden.ts](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Web/hooks/useWisdomGarden.ts) |
| **Android** | `fetchWeek(weekNumber)` ← Refetch ใหม่ทั้งหมดเมื่อ fail ทำให้ item อื่นที่ toggle สำเร็จถูก revert | [NetworkWisdomGardenRepository.kt](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Android/app/src/main/java/com/oatrice/themiddleway/data/repository/NetworkWisdomGardenRepository.kt) |
| **iOS** | `loadWeeklyData(forceRefresh: true)` ← Refetch ใหม่ทั้งหมด เช่นเดียวกับ Android | [WisdomGardenViewModel.swift](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/iOS/TheMiddleWay/Sources/Features/WisdomGarden/ViewModels/WisdomGardenViewModel.swift) |

**วิธีแก้ที่แนะนำ:** Revert เฉพาะ item ที่ fail โดยใช้ `togglePracticeItem(currentData, itemId)` (Web) หรือ toggle `.isCompleted` กลับเฉพาะ item นั้น (Android/iOS)

---

## 🟡 Architectural Issues (ควรปรับปรุง)

### 2. Android: `onStart` ใน Repository ทำให้ Fetch ซ้ำ
- `getWeeklyData()` ใช้ `.onStart` เพื่อ trigger `fetchWeek()` ← อาจทำให้ fetch ซ้ำเมื่อ Compose recompose
- **แนะนำ:** แยก `refreshWeek()` ออกมาเป็น suspend function แยก, ให้ ViewModel เป็นคนเรียก

### 3. Android: FirebaseAuth Listener อาจ Leak
- ใช้ `AuthStateListener` ใน `init` block ← ถ้า `onCleared()` ไม่ถูกเรียก จะ leak
- **แนะนำ:** ย้ายไปใช้ `callbackFlow` ใน Repository จะ cleanup อัตโนมัติ

### 4. iOS: View สร้าง ViewModel เอง (`@StateObject`)
- `WisdomGardenView` สร้าง `@StateObject` เอง ← ถ้า navigate ไป-กลับจะสร้างใหม่ทุกครั้ง
- **แนะนำ:** สร้างใน Parent View แล้วส่งเป็น `@ObservedObject`

### 5. iOS: Repository ไม่แยก Error Type
- catch error แล้ว `throw error` ตรงๆ ← ไม่แยก Offline vs Server Error
- **แนะนำ:** สร้าง `WisdomGardenError` enum (`.offline`, `.serverError`, `.decodingFailed`)

### 6. Backend: `DeleteCategory` ดึงทั้งหมดเพื่อหาตัวเดียว
- เรียก `FindAll` แล้ววนลูปหา category ← ไม่มีประสิทธิภาพ
- **แนะนำ:** เพิ่ม `FindByID` ใน Repository

### 7. Backend: Fallback Query ใน `FindAll`
- ถ้า JOIN ล้มเหลว จะ fallback เป็น query ไม่มี JOIN ← `UsageCount` จะเป็น 0 เสมอ อาจลบ category ที่ยังใช้อยู่ได้

---

## 🧪 Test Suggestions ที่ต้องทำ

### Web
1. ~~Manual Test 1-4 (Offline, Retry, Action Revert, Timeout)~~ → *Manual เท่านั้น*

### Android
1. ~~Manual Test 1-4 (Offline, Retry, Action Revert, Guest Mode, Auth Change)~~ → *Manual เท่านั้น*

### iOS
1. ~~Manual Test 1-4 (Offline, Retry, Action Revert, Guest Mode, Auth Change)~~ → *Manual เท่านั้น*

### Backend (Unit Tests ที่ต้องเขียน)
| # | Test | ไฟล์เป้าหมาย |
|---|------|-------------|
| 1 | `DeleteCategory` กรณี `UsageCount == 0` ⟶ ต้อง return `200 OK` + เรียก `Delete()` | `admin_category_handler_test.go` |
| 2 | `Reorder` กรณี IDs ไม่ตรงกับ DB ⟶ ต้อง fail + ไม่เปลี่ยน order | `admin_category_handler_test.go` |
| 3 | `FindAll` กรณี JOIN fail ⟶ ต้อง return categories มี `UsageCount == 0` | `category_repo_test.go` |

---

## ❓ คำถาม Clarification

1. **Race Condition Fix (Issue #1):** ต้องการให้แก้ทั้ง 3 แพลตฟอร์มเลยไหม? (Web, Android, iOS)
2. **Architectural Issues (#2-#5):** อยาก Refactor ตอนนี้เลยหรือเก็บไว้ทำ Issue ใหม่? (มีผลกระทบค่อนข้างมาก)
3. **Backend Tests (#6-#7):** อยากให้เขียน Unit Test ตาม suggestion ตอนนี้เลยหรือแยกเป็น Issue ใหม่?
4. **`FindByID` ใน Backend:** ต้องการให้สร้าง `FindByID` method ใน repository เลยไหม หรือเฉพาะ test?
