[English](README.md) | [Tiếng Việt](README.vi.md) | [中文](README.zh.md)

# LyricFetcher

[![Build](https://github.com/Shikogoru/lyricfetcher/actions/workflows/ci.yml/badge.svg)](https://github.com/Shikogoru/lyricfetcher/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Shikogoru/lyricfetcher)](go.mod)

**LyricFetcher** là công cụ mạnh mẽ, đa nền tảng với cả hai chế độ: dòng lệnh (**CLI**) hiệu năng cao và giao diện đồ họa (**GUI**) trực quan, tự động tải lời bài hát đồng bộ (`.lrc`) từ [LRCLIB](https://lrclib.net) và [NetEase Cloud Music](https://music.163.com) cho kho nhạc cá nhân của bạn. Ứng dụng được thiết kế tối ưu cho các máy chủ phát nhạc như Navidrome

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


## Cách hoạt động:

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
  - Windows: `tải từ Releases` 


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
⚠️ Tuyên bố miễn trừ trách nhiệm
**NetEase Cloud Music** là một API không chính thức, được sử dụng làm nguồn dự phòng. Tính năng này chỉ dành cho **mục đích cá nhân và học tập**. Chúng tôi không lưu trữ bất kỳ lời bài hát có bản quyền nào. Vui lòng tôn trọng Điều khoản Dịch vụ của NetEase và ủng hộ các nền tảng chính thức. Nhà phát triển LyricFetcher không chịu trách nhiệm cho bất kỳ hành vi sử dụng sai mục đích hoặc bị khóa tài khoản nào do sử dụng tính năng này!

