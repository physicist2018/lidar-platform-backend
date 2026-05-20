# Changelog

## [Unreleased]

### Added
- `plottype="Raw"` в `POST /api/experiments/{id}/image` — возвращает сырой сигнал без `RangeCorrected`/`LogRangeCorrected` трансформации
- Параметры `glueHmin`, `glueHmax` (float64, опциональны) в теле запроса `POST /api/experiments/{id}/image`: задают пользовательский диапазон высот для склейки photon/analog (только с `channelType=glued`)
  - Коэффициент `k` вычисляется по точкам в интервале `[glueHmin, glueHmax]`
  - Выше `glueHmax` → `k*Analog`, ниже `glueHmin` → `Photon`, внутри → среднее
  - Если не указаны — действует прежний алгоритм (1-10 MHz → fallback 5-7 км)
- `GET /api/experiments/{id}/profiles` — список профилей эксперимента (облегчённый, без массивов `Data`/`Altitudes`): поля id, measurementStartTime, measurementStopTime, active, photon, wavelength, polarization, hmin, hmax, bgrType

### Changed
- `gluePairs`: новый алгоритм склейки photon/analog профилей:
  - Коэффициент `k = mean(Photon/Analog)` вычисляется по непрерывному участку высот, где `Photon ∈ [1;10]` MHz и длина ≥ 20 точек; fallback — диапазон высот 5-7 км
  - `Photon > 10 MHz` → `k * Analog` (при `k==0` → `Analog`)
  - `Photon < 1 MHz` → `Photon`
  - `1 ≤ Photon ≤ 10 MHz` → `(Photon + k*Analog) / 2`
- `GetImage`: путь к изображению в MinIO берётся из записи `generated_images.ObjectPath` вместо ручной сборки (устойчивость к изменению формата имени файла)

## [0.6.0] — 2025-07-15

### Added
- Таблица `generated_images` в БД (id, experiment_id, file_name, object_path, wavelength, polarization, channel_type, plot_type, created_at) и миграция
- Доменная сущность `GeneratedImage`
- Порт `GeneratedImageRepository` с методами `Create`, `FindByParams`
- SQLite-адаптер `GeneratedImageRepository` в `internal/infrastructure/repository/sqlite/`
- In-memory-адаптер `InMemoryGeneratedImageRepository` в `internal/infrastructure/repository/`

### Changed
- `GenerateImage`: после загрузки PNG в MinIO сохраняет запись в `generated_images`
- `GetImage`: перед доступом к MinIO проверяет наличие записи в `generated_images`; если нет — возвращает `ErrImageNotGenerated` (404)
- `DownloadImage` хендлер: различает `ErrImageNotGenerated` и прочие ошибки
- `ExperimentUseCase`: добавлено поле `generatedImage`, обновлён конструктор и `main.go`

## [0.5.0] — 2025-07-15

### Added
- `POST /api/experiments/{id}/image` — генерация time-height map (авторизованный):
  - JSON-параметры: `wavelength`, `polarization` (o/p/s), `channelType` (photon/analog/glued), `plottype` (RangeCorrected/LogRangeCorrected)
  - `glued`: склейка photon/analog каналов по времени (photon < 10 МГц → photon, иначе analog)
  - Рендер PNG 1024×768: билинейная интерполяция, jet colormap, подписи осей (время/дистанция), colorbar
  - Сохранение в MinIO по пути `{experimentID}/imgs/time-height_{id}_{wl}_{pol}_{channelType}_{plotType}.png`
  - Зависимость: `golang.org/x/image` (basicfont для подписей)
- `FindByParams` в порту `ExperimentProfileRepository` — фильтрация профилей по experimentID, wavelength, polarization, photon (опционально)

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
