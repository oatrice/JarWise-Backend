# Implementation Plan: Issue #50 — Robust Offline and Server Error Handling

## สรุปปัญหาและเป้าหมาย

แอปปัจจุบันจัดการ error ไม่ดีเมื่อ offline/server ล่ม → silent failure, หน้าว่าง, หรือ error ไม่สื่อ ต้องเพิ่ม:
1. **Full-screen error state** + Retry สำหรับการโหลดข้อมูลหลักที่ล้มเหลว
2. **Toast/Snackbar ชั่วคราว** + UI revert สำหรับ action ที่ล้มเหลว

**ลำดับ:** Android → iOS → Web (ตาม plan.md)

---

## ผลการสำรวจ Codebase (Gap Analysis)

| Platform | สิ่งที่มีอยู่แล้ว | Gap |
|----------|-------------------|-----|
| **Android** | `CoursesUiState` + `LibraryUiState` มี `error: String?` อยู่แล้ว, มี `WisdomGardenEvent.ShowToast`, มี optimistic update ใน `togglePracticeItem` | `WisdomGardenUiState` **ไม่มี `error` field**, `NetworkWisdomGardenRepository.fetchWeek()` **swallow error** (catch แล้ว log เฉยๆ), **ไม่มี `ErrorState` composable**, **ไม่มี `WisdomGardenViewModelTest`** |
| **iOS** | `WisdomGardenViewModel` มี `errorMessage: String?`, มี error handling ใน `loadWeeklyData`, มี optimistic update + revert ใน `toggleItem`, มี tests 2 ไฟล์ | `NetworkWisdomGardenRepository` **catch error แล้วใช้ fallback data** (ไม่ throw ไปถึง ViewModel), **ไม่มี `ErrorStateView`** |
| **Web** | มี `ToasterClient` (sonner) ใน layout.tsx อยู่แล้ว, มี design system (Tailwind + CSS variables), `useWisdomGardenStore` มีอยู่ | Store แทบเปล่า (แค่ `selectedWeek`), **ไม่มี error state**, **ไม่มี `ErrorState` component** |

---

## Proposed Changes

### 1. Android

#### [MODIFY] [WisdomGardenUiState + WisdomGardenViewModel.kt](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Android/app/src/main/java/com/oatrice/themiddleway/ui/wisdomgarden/WisdomGardenViewModel.kt)
- เพิ่ม `error: String? = null` ใน `WisdomGardenUiState`
- ปรับ data collection ใน `init` block ให้จัดการ error case (เมื่อ repository emit `null` + error occurred)
- เพิ่ม `WisdomGardenEvent.ShowSnackbar` สำหรับ toggle failure

#### [MODIFY] [NetworkWisdomGardenRepository.kt](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Android/app/src/main/java/com/oatrice/themiddleway/data/repository/NetworkWisdomGardenRepository.kt)
- `fetchWeek()`: แทนที่จะ swallow exception ให้ throw ไปยัง caller (หรือใช้ error state flow)
- `togglePracticeItem()`: เพิ่ม `WisdomGardenEvent.ShowSnackbar` event เมื่อ toggle ล้มเหลว (ผ่าน callback หรือ SharedFlow)

#### [NEW] [ErrorState.kt](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Android/app/src/main/java/com/oatrice/themiddleway/ui/components/ErrorState.kt)
- Composable ใหม่: icon + ข้อความ error + ปุ่ม "Try Again"
- ใช้ `MaterialTheme.colorScheme.error` เพื่อให้ตรง design system

#### [MODIFY] [WisdomGardenScreen.kt](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Android/app/src/main/java/com/oatrice/themiddleway/ui/wisdomgarden/WisdomGardenScreen.kt)
- เพิ่มเงื่อนไข: ถ้า `uiState.error != null` → แสดง `ErrorState` แทนเนื้อหาปกติ

#### [NEW] [WisdomGardenViewModelTest.kt](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Android/app/src/test/java/com/oatrice/themiddleway/ui/wisdomgarden/WisdomGardenViewModelTest.kt) (TDD)
- 🟥 Test: เมื่อ repository ตอบ error → `uiState.error` ต้องไม่เป็น null
- 🟥 Test: เมื่อ retry สำเร็จ → `uiState.error` ต้องเป็น null + `weeklyData` ต้องไม่เป็น null
- 🟥 Test: เมื่อ toggle ล้มเหลว → event `ShowSnackbar` ถูกส่ง

---

### 2. iOS

#### [MODIFY] [NetworkWisdomGardenRepository.swift](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/iOS/TheMiddleWay/Sources/Features/WisdomGarden/Data/NetworkWisdomGardenRepository.swift)
- `getWeeklyData()`: **ลบ fallback data ใน catch block** → ให้ throw error ไปยัง ViewModel แทน
- แยก error types: `URLError.notConnectedToInternet` → Offline, non-2xx → Server Error

#### [MODIFY] [WisdomGardenViewModel.swift](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/iOS/TheMiddleWay/Sources/Features/WisdomGarden/ViewModels/WisdomGardenViewModel.swift)
- ปรับ `errorMessage` ให้ set ข้อความตาม error type (Offline vs Server error)
- เพิ่ม `isLoading` published property
- ปรับ `toggleItem()` ให้แสดง toast/error message เมื่อ network call ล้มเหลว

#### [NEW] [ErrorStateView.swift](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/iOS/TheMiddleWay/Sources/Features/Shared/ErrorStateView.swift)
- SwiftUI View: VStack ที่มี icon, text, "Retry" button

#### [MODIFY] existing Views ที่แสดง WisdomGarden data ให้ check `errorMessage` แล้วแสดง `ErrorStateView`

#### [MODIFY] [WisdomGardenViewModelTests.swift](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/iOS/TheMiddleWayTests/WisdomGardenViewModelTests.swift) (TDD)
- 🟥 Test: เมื่อ repository throw error → `errorMessage` ต้อง set
- 🟥 Test: retry แล้ว success → `errorMessage` เป็น nil

---

### 3. Web

#### [NEW] [ErrorState.tsx](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Web/components/ui/ErrorState.tsx)
- Reusable component: icon + message + "Retry" button
- ใช้ CSS variables จาก design system (`--error`, `--text-primary`)

#### [MODIFY] [useWisdomGardenStore.ts](file:///Users/oatrice/Software-projects/The%20Middle%20Way%20-Metadata/Platforms/Web/stores/useWisdomGardenStore.ts)
- เพิ่ม `weeklyData`, `isLoading`, `error` state
- เพิ่ม `fetchWeeklyData(week)`, `togglePractice(itemId)` actions
- ใช้ `sonner` `toast.error()` สำหรับ action failure

#### [MODIFY] Wisdom Garden page/components ให้แสดง `ErrorState` เมื่อ error

#### Tests (TDD)
- 🟥 Test ใน vitest: store error state set ถูกต้องเมื่อ fetch ล้มเหลว

---

## Verification Plan

### Automated Tests

| Platform | Command | ที่อยู่ Tests |
|----------|---------|-------------|
| **Android** | `cd Platforms/Android && ./gradlew testDebugUnitTest --tests "*WisdomGardenViewModelTest*"` | `app/src/test/.../WisdomGardenViewModelTest.kt` |
| **iOS** | `cd Platforms/iOS && xcodebuild test -project TheMiddleWay.xcodeproj -scheme TheMiddleWay -destination 'platform=iOS Simulator,name=iPhone 16' -only-testing:TheMiddleWayTests/WisdomGardenViewModelTests` | `TheMiddleWayTests/WisdomGardenViewModelTests.swift` |
| **Web** | `cd Platforms/Web && npx vitest run` | `stores/__tests__/`, `components/__tests__/` |

### Manual Verification (ขอให้ผู้ใช้ช่วยทดสอบ)

เนื่องจาก Feature นี้ต้องทดสอบกับ network conditions จริง (airplane mode, server down) ซึ่งไม่สามารถจำลองผ่าน unit test ได้ 100% ผมจะเขียน code ให้ถูกต้องตาม TDD แต่จะขอให้ผู้ใช้ช่วย test scenarios เหล่านี้บนอุปกรณ์จริง:

1. **Offline Data Load:** เปิด airplane mode → เปิดแอป → ดูว่า error state แสดงไหม
2. **Retry:** ปิด airplane mode → กดปุ่ม Retry → ดูว่าข้อมูลโหลดมาไหม
3. **Offline Action:** เปิด airplane mode → toggle checkbox → ดูว่า snackbar/toast แสดงไหม + checkbox revert ไหม
