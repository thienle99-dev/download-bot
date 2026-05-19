# 📥 Download Bot — Go-based Telegram Downloader

Download Bot là một Telegram Bot viết bằng **Go**, sử dụng `yt-dlp` và `ffmpeg` để tải video YouTube/TikTok (tự động xóa watermark), chuyển đổi MP3 chất lượng cao, lưu trữ lịch sử tải bằng SQLite, lưu cache thông minh và hỗ trợ download file cực lớn vượt giới hạn Telegram một cách mượt mà và thông minh.

---

## 🔥 Chức năng nổi bật
- **Tải YouTube**: Chọn mọi chất lượng video (480p, 720p, 1080p, Best).
- **Tải TikTok**: Tải video nhanh chóng, tự động xóa watermark.
- **Chuyển đổi MP3**: Trích xuất âm thanh từ video với chất lượng tốt nhất.
- **Lịch sử tải**: Xem lại 10 lượt tải gần nhất của bạn bằng lệnh `/history`.
- **Phân phối file thông minh**:
  - **File ≤ 50MB**: Tự động gửi trực tiếp file video/audio vào chat Telegram.
  - **File > 50MB** (Vượt giới hạn mặc định của Telegram Cloud): Bot tự động lưu trên VPS, khởi tạo server HTTP hiệu năng cao cục bộ và gửi **link tải trực tiếp tốc độ cao** ngay trong chat Telegram để người dùng chỉ cần click vào là tải được ngay.
- **Cache thông minh**: Lưu trữ 3 video tải gần nhất trên máy chủ. Gửi lại file tức thì bằng `file_id` Telegram hoặc link HTTP tải nhanh nếu phát hiện trùng lặp, tiết kiệm tối đa băng thông VPS.
- **Inline Mode**: Gõ `@username_bot <url>` trong bất kỳ cuộc hội thoại nào để chọn nhanh định dạng tải.

---

## 🛠️ Yêu cầu hệ thống
- **Docker** & **Docker Compose**
- **Bot Token** (từ [@BotFather](https://t.me/BotFather))

*Đặc biệt: Không cần đăng ký hay sử dụng `TELEGRAM_API_ID` & `TELEGRAM_API_HASH` phức tạp!*

---

## 🚀 Hướng dẫn triển khai nhanh trên VPS

### Bước 1: Clone mã nguồn hoặc chuẩn bị dự án trên VPS
Đảm bảo bạn đã tải thư mục dự án lên VPS.

### Bước 2: Tạo tệp cấu hình `.env`
Sao chép tệp mẫu và điền thông tin của bạn:
```bash
cp .env.example .env
```
Mở tệp `.env` và điền:
- `BOT_TOKEN`: Token bot của bạn từ @BotFather.
- `PUBLIC_URL`: Điền địa chỉ IP Public hoặc tên miền của VPS của bạn kèm cổng (ví dụ: `http://194.156.98.22:8080`). Đây là đường dẫn dùng để sinh link tải trực tiếp khi file > 50MB.

### Bước 3: Khởi tạo và chạy ứng dụng bằng Docker Compose
Dự án được cấu hình cực kỳ tối giản, chạy duy nhất 1 container bao gồm cả Bot và HTTP file server:

```bash
# Build các containers
make build

# Khởi chạy ngầm (Background)
make up
```

### Bước 4: Kiểm tra Logs hệ thống
Đảm bảo bot đã kết nối thành công:
```bash
make logs
```

---

## 📖 Hướng dẫn cấu hình Bot trên Telegram

Để các tính năng hoạt động hoàn hảo nhất, bạn hãy cấu hình Bot qua `@BotFather`:

1. **Enable Inline Mode**:
   - Gửi lệnh `/setinline` tới @BotFather.
   - Chọn Bot của bạn.
   - Nhập mô tả inline ngắn (ví dụ: `Tải video YouTube/TikTok nhanh chóng`).

2. **Set Bot Commands**:
   - Gửi lệnh `/setcommands` tới @BotFather.
   - Chọn Bot của bạn.
   - Nhập danh sách lệnh dưới đây:
     ```text
     start - Bắt đầu sử dụng bot
     help - Hướng dẫn sử dụng
     history - Xem lịch sử tải gần nhất
     ```

---

## 💻 Phát triển cục bộ (Local Development)

Nếu bạn muốn chạy thử nghiệm không cần Docker:

### Yêu cầu:
- Máy tính đã cài đặt **Go (1.22+)**
- Đã cài đặt **yt-dlp** và **ffmpeg** trên biến môi trường hệ thống.

### Khởi chạy:
```bash
# Thiết lập biến môi trường
export BOT_TOKEN="your_bot_token"
export PUBLIC_URL="http://localhost:8080"

# Chạy bot trực tiếp
make dev
```

---

## 📁 Cấu trúc thư mục dự án
```
.
├── cmd/
│   └── bot/
│       └── main.go                 # Điểm khởi chạy hệ thống (Bot + HTTP file server)
├── internal/
│   ├── bot/                        # Code điều khiển logic Telegram Bot
│   │   ├── bot.go                  # Routing & Middleware chính
│   │   ├── handlers.go             # Các lệnh /start, /help, /history
│   │   ├── download_handler.go     # Luồng xử lý tải video & upload (phân phối link > 50MB)
│   │   ├── inline_handler.go       # Tích hợp Inline mode
│   │   ├── callback_handler.go     # Xử lý nút bấm chất lượng tải
│   │   └── keyboards.go           # Cấu hình bàn phím inline
│   ├── downloader/                 # Điều khiển engine yt-dlp + ffmpeg
│   │   ├── downloader.go           # Điều khiển lệnh CLI & progress bar
│   │   └── formats.go              # Định nghĩa cấu hình format
│   ├── storage/                    # Quản lý SQLite (WAL mode, CGO-free)
│   │   ├── sqlite.go               # Thiết lập SQLite & table schema
│   │   ├── history.go              # Các truy vấn CRUD lịch sử
│   │   └── models.go               # Định nghĩa các struct models
│   ├── cache/                      # Hệ thống LRU File Cache
│   │   └── filecache.go            # Cache 3 video/user & tự động xóa đĩa
│   └── config/                     # Trình tải biến môi trường cấu hình
│       └── config.go
├── docker/
│   ├── Dockerfile                  # Multi-stage image build
│   └── docker-compose.yml          # Container Bot + HTTP File Server
├── Makefile                        # Phím tắt điều khiển nhanh
├── go.mod
└── README.md
```

---

## ⚙️ Các lệnh quản lý nhanh (Makefile)
- `make dev`: Chạy thử nghiệm local.
- `make build`: Build lại Docker images.
- `make up`: Khởi chạy container ngầm.
- `make down`: Dừng các containers.
- `make logs`: Xem log bot thời gian thực.
- `make clean`: Dọn dẹp tệp cache cục bộ.
