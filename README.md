# r8-lanshare

A fast, lightweight **Go** CLI utility for two-way local Wi-Fi file transfer between PC and mobile devices using an instant terminal **QR code** — no cables, no cloud, no internet required.

[![Download Release](https://img.shields.io/badge/Download-Linux_%26_Windows-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://github.com/Amin8362/r8-lanshare/releases/latest)

---

## Features

- **Instant Terminal QR Code:** Automatically detects your active LAN IP and renders a crisp terminal QR code for one-scan mobile access.
- **File & Directory Sharing:** Share either a single file or browse an entire folder from any phone browser with dark responsive UI and resume support.
- **Streaming Phone-to-PC Uploads (`-upload`):** Upload multiple large files back to your computer using low-memory multipart streaming with collision-safe filenames.
- **Optional PIN Protection (`-pin`):** Lock access with a custom PIN (`-pin 1234`) or a cryptographically generated 4-digit PIN (`-pin auto`), verified in constant time.
- **Single Static Binary:** Cross-compiled with `CGO_ENABLED=0` and `GOAMD64=v1` for maximum compatibility across Linux and Windows x86-64 machines.

## Usage Examples

    # Share the current folder on port 8080
    ./r8-lanshare

    # Share a single file
    ./r8-lanshare ./movie.mp4

    # Enable two-way transfer (download + upload from phone) with an automatic PIN
    ./r8-lanshare -upload -pin auto ~/Downloads

    # Use a custom port
    ./r8-lanshare -port 9000 -upload .

## Build from Source

    make
    ./r8-lanshare -version

---

<div dir="rtl">

### معرفی فارسی

ابزار **`r8-lanshare`** یک سرور سبک و پرسرعت انتقال فایل روی شبکه محلی (وای‌فای) نوشته‌شده با زبان **Go** است که با تولید **QR کد مستقیم در ترمینال** امکان ارسال و دریافت فایل میان سیستم و گوشی را بدون نیاز به کابل یا اینترنت فراهم می‌کند. قابلیت استریم فایل‌های حجیم، دریافت فایل از گوشی (`-upload`) و قفل رمز عددی (`-pin`) به‌صورت داخلی پیاده‌سازی شده و فایل‌های اجرایی آماده لینوکس و ویندوز در بخش **Releases** قرار دارند.

</div>

---

**Author:** Amin Shirdeli (`R8Force`) • **License:** MIT
