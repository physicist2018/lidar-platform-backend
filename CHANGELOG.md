# Changelog

## [Unreleased]

## [0.4.1] — 2025-07-15

### Added
- Поля `MeasurementStartTime`, `MeasurementStopTime` в доменной сущности `ExperimentProfile` и таблице БД
- Заполнение времён из `LicelFile.MeasurementStartTime` / `MeasurementStopTime` в `Prepare`

## [0.4.0] — 2025-07-15

### Added
- SQLite-адаптеры репозиториев в `internal/infrastructure/repository/sqlite/`:
  - `UserRepository`, `ExperimentRepository`, `ExperimentProfileRepository` — имплементируют порты `ports.*` через `database/sql`
  - `Open(dsn)` — открывает БД, включает WAL + foreign keys, выполняет `CREATE TABLE IF NOT EXISTS`
  - Миграция схемы: таблицы `users`, `experiments`, `experiment_profiles` (индексы по FK)
  - `[]float64` поля (`Altitudes`, `Data`) хранятся как JSON-текст — совместимо с `jsonb` в PostgreSQL
- Поле `DatabaseDSN` в конфигурации (env `DATABASE_DSN`, дефолт `file:lidar.db?cache=shared`)
- `migrations/001_init.sql` — документационная копия DDL
- Зависимость: `modernc.org/sqlite` — pure-Go драйвер SQLite (без CGO)

### Changed
- `main.go`: in-memory репозитории заменены на SQLite, добавлен `defer db.Close()`

### Fixed
- `ExperimentUseCase.setStatus`: теперь read-modify-write (FindByID → Update) вместо перезаписи всей структуры (терялись Title, Comments и др.)

## [0.3.0] — 2025-01-20

### Added
- `POST /api/experiments/{id}/prepare` — подготовка данных эксперимента (авторизованный):
  - Доменная сущность `ExperimentProfile` (ID, ExperimentID, FileName, метаданные канала, обработанные данные Altitudes/Data, Hmin/Hmax, BgrType)
  - Порты: `ExperimentProfileRepository`
  - In-memory репозиторий профилей (`InMemoryExperimentProfileRepository`)
  - `Prepare` в `ExperimentUseCase`:
    - Скачивает zip-архив и опциональный BgrFile из MinIO во временную папку
    - Распаковывает zip через `licelfile.NewLicelPackFromZip`
    - Для каждого канала каждого файла строит массив высот (`altitude[i] = (i + BinShift) * BinWidth`)
    - Вычитает фон: `file` — поканально из загруженного BgrFile, `avgtail` — среднее значений с высоты ≥ BgrAlt
    - Обрезает до `[Hmin, Hmax]` и сохраняет профили
    - Предыдущие профили эксперимента удаляются при повторном вызове
  - JSON-тело запроса: `{"Hmin": <float64>, "Hmax": <float64>, "BgrType": "file"|"avgtail", "BgrAlt": <float64>}` (BgrAlt только для avgtail)
- Метод `Download` в порту `FileStorage` и его реализация в `MinioStorage`

## [0.2.2] — 2025-01-20

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
