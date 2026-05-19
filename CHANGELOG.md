# Changelog

## [Unreleased]

### Added
- `GET /api/experiments` — список экспериментов текущего пользователя (авторизованный)
- `GET /api/experiments/{id}` — получение полной информации об эксперименте (авторизованный)
- `FindByUserID` в интерфейсе `ExperimentRepository` и его in-memory реализация
- `GetByID` и `ListByUser` в `ExperimentUseCase`

## [0.2.1] — 2025-01-20

### Changed
- Загрузка эксперимента теперь асинхронная: `POST /api/experiments` возвращает `201` мгновенно, не дожидаясь завершения
- Статус загрузки отслеживается в поле `Status` модели `Experiment`: `started` → `uploading` → `success` / `failed`
- При ошибке поле `ErrorMessage` содержит причину сбоя

### Added
- `GET /api/experiments/{id}/status` — эндпоинт проверки статуса загрузки (авторизованный)
- Метод `Update` в интерфейсе `ExperimentRepository` и его in-memory реализация
- `ErrExperimentNotFound` — ошибка «эксперимент не найден» в `ExperimentUseCase.GetStatus`

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
