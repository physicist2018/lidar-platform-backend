# Lidar Platform Backend

API для обработки, визуализации и анализа лидарных данных.

## Аутентификация

Все эндпоинты в пространстве `/api/` требуют JWT-токен в заголовке:

```
Authorization: Bearer <token>
```

Токен получается при логине (`POST /auth/login`).

---

## Эндпоинты

### `GET /health`

Проверка работоспособности. Не требует аутентификации.

**Ответ `200 OK`:**

```json
{
  "status": "ok"
}
```

---

### `POST /auth/register`

Регистрация нового пользователя.

**Тело запроса (JSON):**

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

| Поле | Тип | Обязательное |
|------|-----|--------------|
| `email` | string | да |
| `password` | string | да |

**Ответ `201 Created`:**

```json
{
  "id": "uuid-строкой",
  "email": "user@example.com"
}
```

**Ошибки:**

| Код | Причина |
|-----|---------|
| 400 | `email` и/или `password` не указаны |
| 409 | Пользователь с таким email уже существует |

---

### `POST /auth/login`

Вход по email и паролю, получение JWT-токена.

**Тело запроса (JSON):**

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

**Ответ `200 OK`:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Ошибки:**

| Код | Причина |
|-----|---------|
| 400 | `email` и/или `password` не указаны |
| 401 | Неверный email или пароль |

---

### `POST /api/experiments`

Загрузка нового эксперимента. Создаёт запись и запускает асинхронную обработку.

**Тело запроса (multipart/form-data):**

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `title` | string | да | Название эксперимента |
| `comments` | string | да | Комментарий |
| `ZipFile` | file | да | ZIP-архив с licel-файлами |
| `BgrFile` | file | нет | Фоновый licel-файл |
| `MeteoFile` | file | нет | Файл с метеоданными |

Максимальный размер формы — 100 МБ.

**Ответ `201 Created`:**

```json
{
  "id": "uuid-строкой",
  "status": "started"
}
```

Статус обработки отслеживается через `GET /api/experiments/{id}/status`.

**Ошибки:**

| Код | Причина |
|-----|---------|
| 400 | Отсутствуют `title`/`comments`/`ZipFile` или невалидная форма |
| 401 | Не передан токен |

---

### `GET /api/experiments`

Список экспериментов текущего пользователя.

**Ответ `200 OK`:**

```json
[
  {
    "ID": "uuid-строкой",
    "UserID": "uuid-строкой",
    "Title": "Мой эксперимент",
    "Comments": "...",
    "StartDateTime": "2025-01-20T10:30:00Z",
    "StopDateTime": "2025-01-20T11:00:00Z",
    "BgrFilePath": "experiments/{id}/bgr.licel",
    "ZipFilePath": "experiments/{id}/data.zip",
    "MeteoProfilePath": "experiments/{id}/meteo.txt",
    "Status": "success",
    "ErrorMessage": ""
  }
]
```

---

### `GET /api/experiments/{id}`

Получение полной информации об эксперименте по ID.

**Ответ `200 OK`:** (см. структуру выше в `GET /api/experiments`)

**Ошибки:**

| Код | Причина |
|-----|---------|
| 404 | Эксперимент не найден |

---

### `GET /api/experiments/{id}/status`

Статус загрузки/обработки эксперимента.

**Ответ `200 OK`:**

```json
{
  "id": "uuid-строкой",
  "status": "success",
  "error": ""
}
```

Возможные значения `status`:

| Статус | Описание |
|--------|----------|
| `started` | Эксперимент создан, обработка не начата |
| `uploading` | Идёт загрузка файлов в MinIO |
| `success` | Обработка завершена успешно |
| `failed` | Ошибка обработки (причина в поле `error`) |

---

### `GET /api/experiments/{id}/profiles`

Список профилей эксперимента (только метаданные, без массивов данных).

**Ответ `200 OK`:**

```json
[
  {
    "id": "uuid-строкой",
    "measurementStartTime": "2025-01-20T10:30:00Z",
    "measurementStopTime": "2025-01-20T10:30:01Z",
    "active": true,
    "photon": true,
    "wavelength": 532.0,
    "polarization": "o",
    "hmin": 500,
    "hmax": 15000,
    "bgrType": "file"
  }
]
```

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | string | ID профиля |
| `measurementStartTime` | string (RFC 3339) | Время начала измерения |
| `measurementStopTime` | string (RFC 3339) | Время окончания измерения |
| `active` | bool | Признак активного канала |
| `photon` | bool | `true` — photon, `false` — analog |
| `wavelength` | float64 | Длина волны, нм |
| `polarization` | string | Поляризация: `o`, `p`, `s` |
| `hmin` | float64 | Нижняя граница высот после обрезки, м |
| `hmax` | float64 | Верхняя граница высот после обрезки, м |
| `bgrType` | string | Тип вычитания фона: `file` или `avgtail` |

**Ошибки:**

| Код | Причина |
|-----|---------|
| 404 | Эксперимент не найден |

---

### `POST /api/experiments/{id}/prepare`

Подготовка (пересборка) профилей эксперимента — вычитание фона, обрезка по высотам. При повторном вызове предыдущие профили удаляются и создаются заново.

**Тело запроса (JSON):**

```json
{
  "Hmin": 500,
  "Hmax": 15000,
  "BgrType": "avgtail",
  "BgrAlt": 12000
}
```

| Поле | Тип | Обязательное | Описание |
|------|-----|--------------|----------|
| `Hmin` | float64 | да | Нижняя граница высот, м |
| `Hmax` | float64 | да | Верхняя граница высот, м |
| `BgrType` | string | да | `file` — вычитание из фонового файла, `avgtail` — вычитание среднего хвоста |
| `BgrAlt` | float64 | для `avgtail` | Высота, от которой усредняется фоновый хвост, м |

**Ответ `200 OK`:**

```json
{
  "ProfileCount": 1024
}
```

| Поле | Тип | Описание |
|------|-----|----------|
| `ProfileCount` | int | Количество созданных профилей |

**Ошибки:**

| Код | Причина |
|-----|---------|
| 400 | BgrType не `file`/`avgtail`, или для `avgtail` не указан `BgrAlt` |
| 404 | Эксперимент не найден |
| 500 | Ошибка обработки (текст в теле ответа) |

---

### `POST /api/experiments/{id}/image`

Генерация time-height карты (heatmap) в формате PNG. Сохраняется в MinIO, путь доступен для скачивания.

**Тело запроса (JSON):**

```json
{
  "wavelength": 532.0,
  "polarization": "o",
  "channelType": "glued",
  "plottype": "RangeCorrected",
  "glueHmin": 3000,
  "glueHmax": 5000
}
```

| Поле | Тип | Обязательное | Допустимые значения |
|------|-----|--------------|---------------------|
| `wavelength` | float64 | да | Любая длина волны из данных |
| `polarization` | string | да | `o`, `p`, `s` |
| `channelType` | string | да | `photon`, `analog`, `glued` |
| `plottype` | string | да | `Raw`, `RangeCorrected`, `LogRangeCorrected` |
| `glueHmin` | float64 | да (при `channelType=glued`) | Нижняя граница интервала склейки |
| `glueHmax` | float64 | да (при `channelType=glued`) | Верхняя граница интервала склейки |

**Каналы:**

| `channelType` | Описание |
|---------------|----------|
| `photon` | Только счёт фотонов |
| `analog` | Только аналоговый канал |
| `glued` | Склейка photon + analog по алгоритму ниже |

**Типы графиков (`plottype`):**

| `plottype` | Описание |
|-----------|----------|
| `Raw` | Сырой сигнал без трансформации |
| `RangeCorrected` | Сигнал × altitude² (коррекция на дальность) |
| `LogRangeCorrected` | `log₁₀(RangeCorrected)` |

**Алгоритм склейки `glued`:**

1. Коэффициент `k = mean(Photon / Analog)` вычисляется по точкам в интервале высот `[glueHmin, glueHmax]`.
2. Результирующий сигнал для каждой точки:
   - `altitude > glueHmax` → `Photon`
   - `altitude < glueHmin` → `k * Analog`
   - `glueHmin ≤ altitude ≤ glueHmax` → `(Photon + k*Analog) / 2`

**Визуализация:**

Изображение PNG 1024×768 содержит:
- Heatmap с билинейной интерполяцией и jet colormap
- Colorbar справа с подписями тиков
- Сплошные чёрные оси (рамка вокруг области графика)
- Штриховые светло-серые линии сетки по X (время) и Y (высота)
- Подписи осей: «Time» по X (формат `HH:MM`), «Altitude, m» по Y

**Ответ `200 OK`:**

```json
{
  "path": "experiments/{id}/imgs/time-height_532.0_o_glued_RangeCorrected.png"
}
```

Помимо PNG, на MinIO сохраняется JSON с данными для Plot.ly по пути `experiments/{id}/json/time-height_*.json`:

```json
{
  "z": [[...]],
  "y": [1000, 1020, ...],
  "x": ["2025-01-20T15:30:00Z", ...],
  "type": "heatmap"
}
```

**Ошибки:**

| Код | Причина |
|-----|---------|
| 400 | Некорректные значения `polarization`/`channelType`/`plottype`, или `glueHmin`/`glueHmax` не указаны при `channelType=glued`, или указаны без `glued` |
| 500 | Ошибка генерации (например, нет профилей) |

---

### `POST /api/experiments/{id}/profile`

Генерация усреднённого по времени профиля (1D-график) в формате PNG. Те же параметры, что у `/image`, но на выходе — усреднённый по всем профилям сигнал в виде полилайна.

**Тело запроса (JSON):** идентично `POST .../image` — поля `wavelength`, `polarization`, `channelType`, `plottype`, `glueHmin`, `glueHmax`.

**Визуализация:**

Изображение PNG 1024×768 содержит:
- Полилайн (Signal по X, Altitude по Y)
- Сплошные чёрные оси (рамка вокруг области графика)
- Штриховые светло-серые линии сетки по X (Signal) и Y (Altitude)
- Подписи осей: «Signal» по X, «Altitude, m» по Y

**Ответ `200 OK`:**

```json
{
  "path": "experiments/{id}/imgs/profile_532.0_o_glued_RangeCorrected.png"
}
```

Помимо PNG, на MinIO сохраняется JSON с данными для Plot.ly по пути `experiments/{id}/json/profile_*.json`:

```json
{
  "x": [1000, 1020, ...],
  "y": [0.0012, 0.0015, ...],
  "type": "scatter"
}
```

**Ошибки:** аналогичны `POST .../image`.

---

### `GET /api/experiments/{id}/{wavelength}/{polarization}/{channelType}/{plotType}`

Скачивание ранее сгенерированной time-height карты. Значения в URL должны совпадать с параметрами, использованными при генерации.

**Пример:**

```
GET /api/experiments/uuid/532.0/o/glued/RangeCorrected
```

**Ответ `200 OK`:** бинарные данные PNG (`Content-Type: image/png`).

**Ошибки:**

| Код | Причина |
|-----|---------|
| 400 | Не все параметры URL указаны |
| 404 | Изображение не было сгенерировано |

---

## Формат ошибок

Все ошибки возвращаются с HTTP-статусом 4xx/5xx и plain-text сообщением:

```
HTTP/1.1 404 Not Found
Content-Type: text/plain; charset=utf-8

experiment not found
```

---

## Структура проекта

```
├── cmd/
│   └── web/
│       └── main.go                      # Точка входа, DI, запуск HTTP-сервера
├── internal/
│   ├── domain/                          # Бизнес-сущности
│   │   ├── experiment.go                #   Experiment, ExperimentStatus
│   │   ├── experiment_profile.go        #   ExperimentProfile
│   │   ├── generated_image.go           #   GeneratedImage
│   │   └── user.go                      #   User
│   ├── application/
│   │   ├── ports/                       # Порты (интерфейсы)
│   │   │   ├── experiment_repository.go
│   │   │   ├── experiment_profile_repository.go
│   │   │   ├── generated_image_repository.go
│   │   │   ├── file_storage.go
│   │   │   ├── user_repository.go
│   │   │   ├── password_hasher.go
│   │   │   └── token_provider.go
│   │   └── usecases/                    # Бизнес-логика (юзкейсы)
│   │       ├── auth.go                  #   AuthUseCase (Register, Login, Validate)
│   │       ├── experiment.go            #   ExperimentUseCase (Create, Prepare, ListProfiles, ...)
│   │       ├── image.go                 #   Оркестрация генерации heatmap/profile
│   │       ├── glue.go                  #   Склейка photon/analog каналов
│   │       └── profile_processing.go    #   Обработка профилей (матрицы, интерполяция)
│   ├── infrastructure/                  # Адаптеры внешних зависимостей
│   │   ├── auth/
│   │   │   ├── bcrypt.go                #   bcrypt-хешер паролей
│   │   │   └── jwt.go                   #   JWT-провайдер (HS256)
│   │   ├── config/
│   │   │   └── config.go               #   Загрузка конфигурации из переменных окружения
│   │   ├── repository/
│   │   │   ├── sqlite/                  #   SQLite-адаптеры (prod)
│   │   │   │   ├── db.go
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── experiment_repository.go
│   │   │   │   ├── experiment_profile_repository.go
│   │   │   │   └── generated_image_repository.go
│   │   │   ├── inmemory_user.go         #   In-memory адаптеры (для тестов)
│   │   │   ├── inmemory_experiment.go
│   │   │   ├── inmemory_experiment_profile.go
│   │   │   └── inmemory_generated_image.go
│   │   └── storage/
│   │       └── minio.go                 #   MinIO-адаптер (S3-совместимое хранилище)
│   └── interfaces/
│       └── http/                        # HTTP-слой
│           ├── router.go                #   chi-роутер, middleware, регистрация эндпоинтов
│           ├── handlers/
│           │   ├── auth.go              #   POST /auth/register, POST /auth/login
│           │   ├── experiment.go        #   Все эндпоинты экспериментов
│           │   └── health.go            #   GET /health
│           └── middleware/
│               └── auth.go              #   JWT middleware (извлекает UserID в контекст)
├── pkg/
│   └── plot/                            # Утилиты рендеринга графиков
│       ├── plot.go                      #   RenderHeatmap, RenderProfile, DataRange, BilinearInterp
│       ├── colormap.go                  #   JetColormap (jet colormap)
│       └── draw.go                      #   DrawLine, DrawDashedLine, FillRect, DrawString, TextAnchor
├── configs/                             # Конфигурационные файлы
├── migrations/
│   ├── 001_init.sql                     # DDL-миграция (документационная копия)
│   └── 002_add_json_data.sql            # Добавление колонки json_data в generated_images
├── go.mod
├── go.sum
├── AGENTS.md                            # Правила для AI-агентов
├── CHANGELOG.md
└── README.md
```

Архитектура — **гексагональная (Ports & Adapters)**:
- Порты (интерфейсы) определены в `internal/application/ports/`
- Бизнес-логика — в `internal/application/usecases/`, зависит только от портов
- Адаптеры — в `internal/infrastructure/` (БД, MinIO) и `internal/interfaces/` (HTTP)
- Доменные сущности — в `internal/domain/`

---

## Зависимости

| Модуль | Назначение |
|--------|------------|
| `github.com/go-chi/chi/v5` | HTTP-роутер |
| `github.com/golang-jwt/jwt/v5` | JWT-токены (HS256) |
| `github.com/google/uuid` | Генерация UUID |
| `github.com/minio/minio-go/v7` | Клиент MinIO (S3-совместимое хранилище) |
| `github.com/physicist2018/licelfile` | Парсинг licel-файлов |
| `golang.org/x/crypto` | bcrypt-хеширование паролей |
| `golang.org/x/image` | Шрифты для подписей на heatmap (basicfont) |
| `modernc.org/sqlite` | Pure-Go драйвер SQLite (без CGO) |

---

## Запуск

### Требования

- Go 1.25+
- Доступный MinIO (или S3-совместимый) сервер

### Конфигурация

Создать файл `.env` (или задать переменные в окружении):

```bash
export JWT_SECRET="your-secret-key-at-least-32-chars"
export JWT_EXPIRY="24h"
export DATABASE_DSN="file:lidar.db?cache=shared"
export MINIO_ENDPOINT="localhost:9000"
export MINIO_ACCESS_KEY="minioadmin"
export MINIO_SECRET_KEY="minioadmin"
export MINIO_BUCKET="lidar"
export MINIO_USE_SSL="false"
export PORT="8080"
```

### Сборка и запуск

```bash
# Установка зависимостей
go mod download

# Сборка
go build -o lidar-backend ./cmd/web/

# Запуск
./lidar-backend
```

Или одной командой:

```bash
go run ./cmd/web/
```

Сервер стартует на адресе из `$HOST:$PORT` (по умолчанию `0.0.0.0:8080`).

### Проверка работоспособности

```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### Типовой сценарий использования

```bash
# 1. Регистрация
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}'

# 2. Логин (получить токен)
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret123"}' | jq -r '.token')

# 3. Загрузка эксперимента
EXP_ID=$(curl -s -X POST http://localhost:8080/api/experiments \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Тестовый эксперимент" \
  -F "comments=Проверка API" \
  -F "ZipFile=@data.zip" \
  -F "BgrFile=@background.licel" | jq -r '.id')

# 4. Проверка статуса
curl http://localhost:8080/api/experiments/$EXP_ID/status \
  -H "Authorization: Bearer $TOKEN"

# 5. Подготовка профилей
curl -X POST http://localhost:8080/api/experiments/$EXP_ID/prepare \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"Hmin":500,"Hmax":15000,"BgrType":"avgtail","BgrAlt":12000}'

# 6. Список профилей
curl http://localhost:8080/api/experiments/$EXP_ID/profiles \
  -H "Authorization: Bearer $TOKEN"

# 7. Генерация heatmap
curl -X POST http://localhost:8080/api/experiments/$EXP_ID/image \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"wavelength":532.0,"polarization":"o","channelType":"glued","plottype":"RangeCorrected"}'

# 8. Скачивание heatmap
curl http://localhost:8080/api/experiments/$EXP_ID/532.0/o/glued/RangeCorrected \
  -H "Authorization: Bearer $TOKEN" \
  -o heatmap.png
```

Приложение конфигурируется через переменные окружения:

| Переменная | Назначение | По умолчанию |
|------------|------------|--------------|
| `PORT` | Порт HTTP-сервера | `8080` |
| `HOST` | Хост для прослушивания | `0.0.0.0` |
| `JWT_SECRET` | Секретный ключ для подписи JWT | *обязательная* |
| `JWT_EXPIRY` | Срок жизни токена | `24h` |
| `DATABASE_DSN` | DSN для SQLite | `file:lidar.db?cache=shared` |
| `MINIO_ENDPOINT` | Адрес MinIO | *обязательная* |
| `MINIO_ACCESS_KEY` | Ключ доступа MinIO | *обязательная* |
| `MINIO_SECRET_KEY` | Секретный ключ MinIO | *обязательная* |
| `MINIO_BUCKET` | Имя бакета | *обязательная* |
| `MINIO_USE_SSL` | Использовать SSL для MinIO | `false` |
| `READ_TIMEOUT` | Таймаут чтения запроса | `30s` |
| `WRITE_TIMEOUT` | Таймаут записи ответа | `30s` |
| `IDLE_TIMEOUT` | Таймаут keep-alive | `60s` |
| `MAX_HEADER_BYTES` | Макс. размер заголовков | `1MB` |
