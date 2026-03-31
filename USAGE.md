# LogT — Руководство по использованию

## Быстрый старт

### Установка

```bash
# Из исходников
git clone https://github.com/turkprogrammer/logt.git
cd logt && go build -o logt ./cmd/logt

# Установка в GOPATH/bin
go install ./cmd/logt

# Локальная сборка (если репозиторий не опубликован)
go build -o logt ./cmd/logt
```

### Запуск

```bash
# Следить за файлом
logt ./app.log

# Несколько файлов по маске
logt ./logs/*.log

# Мульти-маска (разные директории)
logt ./api/*.log ./db/*.log ./cache/*.log

# Из stdin
cat app.log | logt

# С фильтром по уровню
logt --level error ./app.log

# С фильтром по времени
logt --since 1h ./app.log
logt --since 30m --until 10m ./app.log
logt --since "2024-01-15 10:00" ./app.log
```

---

## Горячие клавиши

| Клавиша | Действие |
|---------|----------|
| `Space` | Пауза / Продолжить |
| `/` | Открыть Fuzzy фильтр |
| `r` | Переключить Fuzzy ↔ Regex |
| `Enter` | Применить фильтр / Открыть JSON |
| `Backspace` | Удалить символ |
| `Esc` | Очистить фильтр |
| `↑ / ↓` | Прокрутка |
| `PgUp / PgDn` | Прокрутка по страницам |
| `Home / End` | Начало / Конец |
| `g` | В начало (less) |
| `G` | В конец (less) |
| `n / N` | Следующее / Предыдущее совпадение |
| `Tab` | Показать / скрыть панель источников |
| `q` | Выход |

---

## Фильтрация по времени

LogT поддерживает фильтрацию логов по временным меткам с помощью флагов `--since` и `--until`.

### Форматы времени

**Относительное время** (длительность):
```bash
logt --since 1h ./app.log      # Логи за последний час
logt --since 30m ./app.log     # Логи за последние 30 минут
logt --since 24h ./app.log     # Логи за последние 24 часа
logt --since 1h30m ./app.log   # Логи за последние 1.5 часа
```

**Абсолютное время** (конкретная дата/время):
```bash
logt --since "2024-01-15 10:00" ./app.log    # С 10 утра 15 января
logt --since "2024-01-15" ./app.log          # С начала дня 15 января
logt --since 2024-01-15T10:00:00 ./app.log   # ISO8601 формат
```

**Диапазон времени**:
```bash
logt --since 1h --until 10m ./app.log        # Логи между 1 часом и 10 минутами назад
logt --since "2024-01-15 10:00" --until "2024-01-15 12:00" ./app.log
```

### Комбинация с другими фильтрами

```bash
# Только ошибки за последний час
logt --since 1h --level error ./app.log

# Только WARN и ERROR за последние 30 минут
logt --since 30m --level warn ./app.log

# Текстовый поиск + время
logt --since 1h "connection failed" ./app.log
```

---

## JSON Path фильтрация

LogT поддерживает мощную JSON Path фильтрацию для работы с JSON логами, похожую на `jq`.

### Базовый синтаксис

**Точное совпадение** (`==`):
```bash
# Только ошибки
logt --json '.level == "error"' ./app.log

# Только определённый сервис
logt --json '.service == "api-gateway"' ./app.log

# Числовые значения
logt --json '.status == 500' ./app.log

# Boolean значения
logt --json '.success == false' ./app.log
```

**Отрицание** (`!=`):
```bash
# Всё кроме debug
logt --json '.level != "debug"' ./app.log

# Всё кроме статуса 200
logt --json '.status != 200' ./app.log
```

**Префикс** (`startswith`):
```bash
# Сообщения начинающиеся с "Error"
logt --json '.message | startswith("Error")' ./app.log

# URL начинающиеся с /api
logt --json '.url | startswith("/api")' ./app.log
```

**Подстрока** (`contains`):
```bash
# Сообщения содержащие "timeout"
logt --json '.message | contains("timeout")' ./app.log

# Ошибки подключения
logt --json '.error | contains("connection")' ./app.log
```

### Вложенные поля

```bash
# Фильтрация по вложенному полю
logt --json '.user.role == "admin"' ./app.log

# Глубокая вложенность
logt --json '.request.headers.authorization | startswith("Bearer")' ./app.log
```

### Комбинация с другими фильтрами

```bash
# JSON Path + время
logt --json '.level == "error"' --since 1h ./app.log

# JSON Path + текстовый поиск
logt --json '.level == "error"' "stacktrace" ./app.log

# JSON Path + уровень + время
logt --json '.service == "api"' --level error --since 30m ./app.log
```

### В конфигурации

**YAML** (`~/.config/logt/config.yaml`):
```yaml
json-filter: '.level == "error"'
```

**Переменные окружения**:
```bash
export LOGT_JSON_FILTER='.level == "error"'
logt ./app.log
```

---

## В конфигурации

**YAML** (`~/.config/logt/config.yaml`):
```yaml
since: 1h
until: 10m
```

**Переменные окружения**:
```bash
export LOGT_SINCE=1h
export LOGT_UNTIL=10m
logt ./app.log
```

---

## Кейсы применения

### 1. Поиск ошибок пользователя

```bash
logt ./app.log
# Нажмите / → введите user_id:123 → Enter
```

### 2. Анализ JSON логов

```bash
logt ./app.log
# Выберите JSON строку → Enter для просмотра дерева
```

### 3. Мониторинг в реальном времени

```bash
logt ./logs/*.log
# Смотрите цветовые паттерны: синий=INFO, жёлтый=WARN, красный=ERROR
```

### 4. Экспорт отфильтрованных логов

```bash
# В файл
logt --forward errors.log ./app.log

# В stdout (pipe)
logt --forward - ./app.log | grep ERROR
```

### 5. Отладка CI/CD пайплайнов

```bash
kubectl logs -f pod/myapp | logt
journalctl -f | logt
docker logs -f mycontainer | logt
```

---

## Конфигурация

### YAML файл (`~/.config/logt/config.yaml`)

```yaml
buffer-size: 5000
buffer-max: 10000
theme: dark
forward: filtered.log
sources:
  - /var/log/*.log
since: 1h
until: 10m
```

### Переменные окружения

```bash
export LOGT_BUFFER_SIZE=10000
export LOGT_LEVEL=error
export LOGT_FORWARD=out.log
export LOGT_SINCE=1h
export LOGT_UNTIL=10m
export LOGT_JSON_FILTER='.level == "error"'
```

### Флаги командной строки

| Флаг | Короткий | Описание |
|------|----------|----------|
| `--path` | `-p` | Пути к файлам или glob паттерны |
| `--level` | `-l` | Фильтр по уровню (debug,info,warn,error) |
| `--buffer` | `-b` | Размер буфера (по умолчанию 5000) |
| `--max-buffer` | `-m` | Максимальный размер буфера |
| `--theme` | `-t` | Тема (dark, light) |
| `--forward` | `-f` | Экспорт логов (файл или stdout) |
| `--since` | `-S` | Фильтр с времени (1h, 30m, 2024-01-15) |
| `--until` | `-U` | Фильтр по время (1h, 30m, 2024-01-15) |
| `--json` | `-j` | JSON Path фильтр (например: `.level == "error"`) |
| `--version` | `-v` | Показать версию |
| `--help` | `-h` | Показать помощь |

---

## Форматы логов

**Автоопределение:** JSON, Logfmt, Plain

**Уровни:** FATAL, ERROR, WARN, INFO, DEBUG, TRACE (без учёта регистра)

**Примеры:**
```json
{"level": "error", "msg": "Connection failed", "host": "db-01"}
```
```logfmt
level=error msg="Connection failed" host=db-01 port=5432
```
```
2024-01-15 10:30:00 ERROR Connection failed
```
