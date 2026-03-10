# 📋 สรุปการวิเคราะห์ Issue #50: Robust Offline and Server Error Handling

> อ่านไฟล์ทั้งหมด 8 ไฟล์เรียบร้อยแล้ว

---

## 🔍 สรุปภาพรวม Feature

**ปัญหาปัจจุบัน:** แอปไม่มีการจัดการ error ที่ดีเมื่อ offline หรือ server ไม่ตอบ → ทำให้เกิด silent failure / หน้าจอเปล่า / ข้อความ error ที่ไม่สื่อความหมาย

**สิ่งที่ต้องทำ (2 กลุ่มหลัก):**

| กลุ่ม | เมื่อไหร่ | UI Response |
|-------|----------|-------------|
| **FR-1: Data Fetch Error** | โหลดหน้าจอแล้ว API ล้มเหลว | **Full-screen error state** + ปุ่ม Retry |
| **FR-2: Action/Mutation Error** | ผู้ใช้ทำ action (เช่น toggle checkbox) แล้วล้มเหลว | **Toast/Snackbar** ชั่วคราว + revert สถานะ UI |

**ลำดับพัฒนา:** Android → iOS → Web

**แต่ละ platform จะสร้าง:**
- `NetworkError` sealed class/enum (`Offline`, `Server`, `Generic`)
- `ErrorState` component (full-screen + retry button)
- ปรับ Repository ให้ return `Result<T, Error>` แทนการ throw
- ปรับ ViewModel ให้มี `error` + `isLoading` state

---

## ❓ คำถามเพื่อ Clarify ก่อน Implement

1. **เริ่มจากแพลตฟอร์มไหนก่อน?** Plan ระบุว่า Android FIRST แต่ในบริบทของ session นี้ คุณต้องการให้เริ่มจากแพลตฟอร์มไหนก่อน? (Android / iOS / Web / Backend?) — หรือทำเฉพาะแพลตฟอร์มเดียวก่อน?

2. **Backend (Go) ต้อง implement อะไรไหม?** ใน `prompt_backend.txt` ถูก generate มาด้วย แต่ `analysis.md` ระบุว่า Backend impact เป็น 🟢 Low — Backend ส่ง HTTP status code ถูกต้องอยู่แล้วหรือยัง?

3. **โครงสร้าง Codebase ปัจจุบัน:** ยังไม่ได้สำรวจโครงสร้างโค้ดจริง — ต้องการให้สำรวจก่อนไหม?

4. **Design System / Theme:** มี design system หรือ color palette ที่ใช้อยู่แล้วไหม?

5. **ข้อความ error ข้ามแพลตฟอร์ม** ต้องใช้ตาม SBE เป๊ะๆ หรือปรับได้?

6. **Optimistic UI revert**: ปัจจุบัน code ทำ optimistic update อยู่แล้วหรือเปล่า?
