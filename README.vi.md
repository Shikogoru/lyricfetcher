[English](README.md) | [Tiếng Việt](README.vi.md) | [中文](README.zh.md)

# LyricFetcher

[![Build](https://github.com/Shikogoru/lyricfetcher/actions/workflows/ci.yml/badge.svg)](https://github.com/Shikogoru/lyricfetcher/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Shikogoru/lyricfetcher)](go.mod)

**LyricFetcher** là công cụ mạnh mẽ, đa nền tảng với cả hai chế độ: dòng lệnh (**CLI**) hiệu năng cao và giao diện đồ họa (**GUI**) trực quan, tự động tải lời bài hát đồng bộ (`.lrc`) từ [LRCLIB](https://lrclib.net) và [NetEase Cloud Music](https://music.163.com) cho kho nhạc cá nhân của bạn. Ứng dụng được thiết kế tối ưu cho các máy chủ phát nhạc như **Navidrome**, **Feishin** và **Symfonium**.

---

## Giao diện đồ họa (GUI)

LyricFetcher tích hợp sẵn giao diện người dùng trực quan xây dựng trên thư viện [Fyne](https://fyne.io):

- **Chọn thư mục dễ dàng**: Hỗ trợ hộp thoại duyệt thư mục hoặc nhập đường dẫn trực tiếp.
- **Tùy chọn thu gọn linh hoạt**: Bật/tắt ghi đè, chế độ khớp nghiêm ngặt, tắt nguồn phụ NetEase và thanh kéo chỉnh số worker (1-16).
- **Tiến trình & Thống kê thời gian thực**: Thanh tiến trình mượt mà, hiển thị tên tệp đang xử lý và 4 chỉ số thống kê tức thì (Đã tìm, Không thấy, Bỏ qua, Lỗi).
- **Nút Bắt đầu / Dừng an toàn**: Chủ động bắt đầu hoặc hủy luồng xử lý bất cứ lúc nào qua Context cancel.
- **Chuyển đổi ngôn ngữ tức thì**: Chuyển đổi giữa Tiếng Việt, English và 中文 ngay trên góc phải cửa sổ mà không cần khởi động lại.
- **Nút mở thư mục một chạm**: Mở ngay thư mục nhạc trên File Explorer sau khi hoàn tất.

Khởi chạy GUI:
```bash
./bin/lyricfetcher-gui
# Trên Windows:
.\bin\lyricfetcher-gui.exe
```

---

## Tính năng nổi bật

- **Đọc Metadata đa định dạng**: Trích xuất thẻ metadata từ MP3 (ID3v2), FLAC (Vorbis Comment) và M4A/MP4 (tên bài, nghệ sĩ, album, thời lượng).
- **Tự động chuẩn hóa Metadata**: Chuẩn hóa Unicode NFC, tự động tách chuỗi "Tiêu đề - Nghệ sĩ" bằng dấu gạch ngang, lọc bỏ hậu tố nghệ sĩ trùng lặp.
- **Chuỗi Fallback thông minh 4 tầng**:
  - Tầng 1: Tên bài + Nghệ sĩ + Album + Thời lượng (khớp chính xác)
  - Tầng 2: Tên bài + Nghệ sĩ
  - Tầng 3: Tìm kiếm linh hoạt
  - Tầng 4: Nguồn dự phòng NetEase Cloud Music
- **Worker Pool hiệu năng cao**: Tùy chỉnh số lượng worker chạy đồng thời (`--workers 1-16`) được kiểm soát bởi Rate Limiter chuẩn xác (LRCLIB: 4 req/giây, NetEase: 1 req/giây).
- **Bộ nhớ đệm thông minh & Cơ chế Resume**:
  - Tự động lưu cache kết quả với khóa SHA-256 trong thư mục cấu hình hệ thống.
  - Bỏ qua tức thì các file đã có `.lrc` tương ứng khi quét lại, không tốn tài nguyên mạng.
- **Hỗ trợ đa ngôn ngữ (i18n)**: Giao diện và thông báo bằng Tiếng Anh (`en`), Tiếng Việt (`vi`) và Tiếng Trung (`zh`).
- **Thanh tiến trình TTY realtime & Ghi nhật ký chuẩn ISO**: Hiển thị tiến trình mượt mà trên Terminal và ghi file log chi tiết theo định dạng ISO 8601.

---

## Bắt đầu nhanh

```bash
# Tải lời bài hát cho 1 tệp đơn lẻ
lyricfetcher "01. Drive By.mp3"

# Quét đệ quy toàn bộ thư mục nhạc với 4 workers mặc định
lyricfetcher "/path/to/Music"

# Chạy thử mô phỏng (dry-run) với giao diện tiếng Việt
lyricfetcher --lang vi --dry-run "/path/to/Music"
```

---

## Luồng hoạt động

```text
  Tệp âm thanh cục bộ (mp3 / flac / m4a)
                 │
                 ▼
     [1. Đọc Metadata tệp nhạc] ──> Tên bài, Nghệ sĩ, Album, Thời lượng
                 │
                 ▼
       [2. Kiểm tra bộ nhớ đệm] ──(Trúng)──> Ghi file .lrc
                 │ (Trượt)
                 ▼
       [3. Truy vấn API LRCLIB] ──(Tìm thấy)──> Lưu Cache & Ghi .lrc
                 │ (Không có)
                 ▼
     [4. Dự phòng API NetEase]  ──(Tìm thấy)──> Lưu Cache & Ghi .lrc
                 │ (Không có)
                 ▼
       Ghi nhận [Không tìm thấy]
```

---

## Cài đặt

### Khuyến nghị: Bản build sẵn (Pre-Built Binaries)
Đa số người dùng **không cần** cài đặt Go hay GCC. Bạn chỉ cần tải file thực thi sẵn sàng sử dụng từ trang [GitHub Releases](https://github.com/Shikogoru/lyricfetcher/releases):
- **Bản GUI (Giao diện đồ họa trực quan)**:
  - Windows: `lyricfetcher-gui-windows-amd64.exe` (đã nhúng sẵn bộ font Google Noto Sans & Noto Sans CJK SC hiển thị chuẩn Tiếng Trung/Nhật/Hàn)
- **Bản CLI (Dòng lệnh)**:
  - Windows: `lyricfetcher-windows-amd64.exe` hoặc `windows-arm64.exe`
  - Linux: `lyricfetcher-linux-amd64` hoặc `linux-arm64`
  - macOS: `lyricfetcher-darwin-amd64` hoặc `darwin-arm64`

### Biên dịch từ mã nguồn
Nếu bạn muốn tự biên dịch:
- **Chỉ bản CLI (Pure Go, không yêu cầu CGO hay GCC)**:
  ```bash
  git clone https://github.com/Shikogoru/lyricfetcher.git
  cd lyricfetcher
  go build -o bin/lyricfetcher ./cmd/cli
  ```
- **Bản GUI (Yêu cầu CGO & GCC)**:
  Biên dịch giao diện Fyne yêu cầu bật `CGO_ENABLED=1` và cài đặt trình biên dịch C (GCC qua [w64devkit](https://github.com/skeeto/w64devkit) hoặc MinGW-w64 trên Windows; `gcc` / `clang` trên Linux/macOS):
  ```bash
  # Trên Windows PowerShell:
  .\build.ps1 -Target build

  # Trên Linux / macOS:
  make build
  ```

---

## Bảng tùy chọn lệnh (CLI Options)

| Nhóm | Tùy chọn (Flag) | Mặc định | Mô tả chi tiết |
| :--- | :--- | :--- | :--- |
| **Chung** | `--lang` | `auto` | Ngôn ngữ giao diện: `en`, `vi`, `zh`, `auto` |
| | `--version` | `false` | In thông tin phiên bản và thoát |
| | `--help` | `false` | Hiển thị màn hình trợ giúp đầy đủ |
| **Xử lý** | `--workers` | `4` | Số lượng luồng worker chạy song song (1 - 16) |
| | `--strict` | `false` | Chế độ nghiêm ngặt: yêu cầu thời lượng khớp trong khoảng +/- 2s |
| | `--no-netease` | `false` | Tắt nguồn dự phòng NetEase |
| | `--netease-spoof-ip` | `false` | Bật header giả lập IP cho NetEase |
| | `--netease-spoof-ip-value` | `118.89.204.198` | Địa chỉ IP gửi qua header `X-Real-IP` của NetEase |
| **Đầu ra** | `--output-dir` | `""` | Thư mục lưu file `.lrc` (mặc định: cùng thư mục với file nhạc) |
| | `--overwrite` | `false` | Ghi đè file `.lrc` nếu đã tồn tại |
| | `--dry-run` | `false` | Mô phỏng thực thi, không tạo hoặc sửa file thật |
| | `--log-file` | `lyricfetcher.log` | Đường dẫn ghi file nhật ký chi tiết |
| **Cache** | `--no-cache` | `false` | Tắt hoàn toàn việc đọc/ghi bộ nhớ đệm |
| | `--clear-cache` | `false` | Xóa sạch bộ nhớ đệm trước khi chạy |

---

## Ví dụ thực tế

### 1. Tải lời bài hát cho 1 bài duy nhất
```bash
lyricfetcher "California 37/01. Drive By.mp3"
```

### 2. Quét nhanh kho nhạc lớn với 8 workers
```bash
lyricfetcher --workers 8 "/media/music"
```

### 3. Khớp chính xác thời lượng (Tránh nhầm bản Live/Remix)
```bash
lyricfetcher --strict "/media/music"
```

### 4. Xuất toàn bộ file lời bài hát ra thư mục riêng
```bash
lyricfetcher --output-dir "/media/lyrics" "/media/music"
```

### 5. Xóa cache và tải lại mới toàn bộ
```bash
lyricfetcher --clear-cache --overwrite "/media/music"
```

---

## Tương thích với Navidrome và các ứng dụng nghe nhạc

LyricFetcher tạo các file lời bài hát chuẩn `.lrc` đặt cạnh tệp âm thanh gốc theo định dạng `<tên-bài-hát>.lrc`.

> [!IMPORTANT]
> **Chạy Full Scan trong Navidrome**: Tính năng quét nhanh (Quick Scan) định kỳ của Navidrome chỉ phát hiện các bài hát mới thêm vào, **không** tự động quét các file `.lrc` mới tạo cạnh bài hát cũ. Sau khi chạy LyricFetcher, bạn hãy vào Navidrome (`Settings` -> `Activity` -> bấm **Start Full Scan**) để máy chủ lập chỉ mục lại toàn bộ file lời bài hát.

Tương thích tốt với:
- **Giao diện Web Navidrome**: Hiển thị nút biểu tượng lời bài hát trên thanh phát nhạc.
- **Feishin (Máy tính)**: Bảng cuộn lời bài hát tương tác theo thời gian thực.
- **Symfonium (Android)**: Hiển thị lời karaoke đồng bộ và cho phép chỉnh lệch thời gian (offset).

Xem thêm [Hướng dẫn tích hợp Navidrome](examples/navidrome-workflow.md).

---

