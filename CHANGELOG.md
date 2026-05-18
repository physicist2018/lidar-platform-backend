# Changelog

## [0.2.0] — 2025-01-20

### Added
- Загрузка эксперимента (`POST /api/experiments`, авторизованный):
  - Доменная сущность `Experiment` (ID, UserID, Title, Comments, StartDateTime, StopDateTime, BgrFilePath, ZipFilePath, MeteoProfilePath)
  - Порты: `ExperimentRepository`, `FileStorage`
  - `ExperimentUseCase` — извлекает StartDateTime/StopDateTime из licel-файлов внутри zip, сохраняет файлы в MinIO
  - In-memory репозиторий экспериментов
  - MinIO-адаптер `FileStorage` (автосоздание бакета)
  - Multipart/form-data хендлер: обязательные title/comments/ZipFile, опциональные BgrFile/MeteoFile
  - Конфигурация MinIO (`MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`)
- Зависимости: `licelfile` (извлечение MeasurementStartTime/StopTime), `minio-go/v7`

## [0.1.0] — 2025-01-20

### Added
- Инициализация проекта (Go module `github.com/lidar-platform/backend`, каноническая структура по гексагональной архитектуре)
- HTTP-роутер на `chi` с базовыми middleware (Logger, Recoverer, RequestID, RealIP, Heartbeat)
- `GET /health` — проверка работоспособности
- Конфигурация из переменных окружения (`PORT`, `HOST`, `READ_TIMEOUT`, `WRITE_TIMEOUT`, `IDLE_TIMEOUT`, `MAX_HEADER_BYTES`)
- Graceful shutdown по SIGINT/SIGTERM
- Аутентификация пользователей:
  - Доменная сущность `User` (ID, Email, PasswordHash)
  - Порты: `UserRepository`, `PasswordHasher`, `TokenProvider`
  - `AuthUseCase` — Register, Login, Validate
  - bcrypt-хеширование паролей
  - JWT-токены (HS256) с настраиваемым сроком жизни (`JWT_SECRET`, `JWT_EXPIRY`)
  - In-memory репозиторий пользователей
  - `POST /auth/register` — регистрация
  - `POST /auth/login` — вход, возврат JWT
  - JWT middleware для защищённых маршрутов (извлекает `Authorization: Bearer <token>` в `context`)
