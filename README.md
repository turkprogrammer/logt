# LogT — Современный Explorer логов (TUI)

> **Легковесная альтернатива lnav** с упором на UX, авто-парсинг JSON и мгновенную фильтрацию.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge)
![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)
[![Tests](https://img.shields.io/badge/Tests-36%20passing-44b526?style=for-the-badge)]()

## 🚀 Возможности

### Основные
- **Мульти-source tailing** — Слежение за несколькими файлами по шаблону (`./logs/*.log`)
- **Ring Buffer** — Хранит последние 5000 строк в памяти (настраивается)
- **Stdin Support** — `cat app.log | logt`
- **Log Forwarding** — Экспорт отфильтрованных логов в файл (`--forward`) или stdout (`--forward -`)

### Просмотр
- **Live Tail** — Автопрокрутка при поступлении новых строк
- **Подсветка синтаксиса** — Цветовая кодировка уровней (INFO=синий, WARN=желтый, ERROR=красный)
- **JSON Expand** — Нажмите Enter на JSON строке для разворачивания в полноэкранное дерево

### Интерактивность
- **Fuzzy Filter** — Нажмите `/` для мгновенной фильтрации
- **Regex Filter** — Нажмите `r` для режима регулярных выражений
- **Pause/Resume** — Нажмите `Space` для паузы автопрокрутки
- **Source Toggle** — Нажмите `Tab` для показа/скрытия панели источников
- **JSON Explorer** — Нажмите `Enter` на JSON строке для просмотра ключей

## 🚀 Тесты производительности

Протестировано на **Intel Core i3-10100 @ 3.6 GHz**:

| Операция | Скорость | Примечания |
|-----------|-------|-------|
| RingBuffer Add | **~27M ops/sec** | Потокобезопасный, блокировка-free чтение |
| JSON Парсинг | **~314K строк/sec** | Авто-детект + парсинг |
| Fuzzy Фильтр | **~3.7M совпадений/sec** | Без учёта регистра |
| Определение уровня | **~394K строк/sec** | На основе регулярных выражений |
| IsValidJSON | **~1.3M проверок/sec** | Быстрая валидация |

**Память**: Ring buffer ограничен ~2MB для 5000 строк (настраивается)

*LogT разработан для высоконагруженных окружений с минимальным потреблением CPU/RAM.*

## 📦 Установка

### Из исходников
```bash
# Клонировать репозиторий
git clone https://github.com/turkprogrammer/logt.git
cd logt

# Собрать бинарник
go build -o logt ./cmd/logt

# Или установить в GOPATH
go install ./cmd/logt
```

### Предсобранные бинарники
Скачать с [Releases](https://github.com/turkprogrammer/logt/releases)

> **Примечание:** Если репозиторий ещё не опубликован, используйте локальную разработку:
> ```bash
> go build -o logt ./cmd/logt
> ```

## 🛠️ Использование

### Базовое
```bash
# Следить за одним файлом
logt ./app.log

# Следить за несколькими файлами по шаблону
logt ./logs/*.log

# Фильтр по уровню
logt --level error ./app.log

# Stdin pipe
cat app.log | logt
```

### Тестовые логи

#### Windows (PowerShell)
```powershell
# Простой JSON (с кавычками)
echo '{"level":"info","msg":"Server started"}' | Out-File -FilePath app.log -Encoding utf8

# JSON с доп. полями
echo '{"level":"error","msg":"Connection failed","host":"db1","port":5432}' | Out-File app.log -Encoding utf8

# Несколько строк
@'
{"level":"info","msg":"Starting"}
{"level":"warn","msg":"Low memory"}
{"level":"error","msg":"OOM killed"}
{"level":"debug","msg":"GC done"}
'@ | Out-File app.log -Encoding utf8

# Запуск
.\logt.exe app.log

# Pipe
Get-Content app.log | .\logt.exe
```

#### Windows (CMD)
```cmd
# Простой JSON
echo {"level":"info","msg":"Server started"} > app.log

# JSON с доп. полями - используем printf для кавычек
printf "{\"level\":\"error\",\"msg\":\"Connection failed\",\"host\":\"db1\",\"port\":5432}\n" > app.log

# Несколько строк (CMD)
(
  echo {"level":"info","msg":"Starting"}
  echo {"level":"warn","msg":"Low memory"}
  echo {"level":"error","msg":"OOM killed"}
  echo {"level":"debug","msg":"GC done"}
) > app.log

# Запуск
logt.exe app.log

# Pipe
type app.log | logt.exe
```

#### Linux / macOS
```bash
# Простой JSON
echo '{"level":"info","msg":"Server started"}' > app.log

# JSON с доп. полями
echo '{"level":"error","msg":"Connection failed","host":"db1","port":5432}' > app.log

# Logfmt
echo 'level=info msg="Server started"' > app.log
echo 'level=error msg="Connection failed" host=db1 port=5432' > app.log

# Plain text с уровнем
echo '2024-01-15 10:30:00 INFO Server started' > app.log
echo '2024-01-15 10:30:01 ERROR Connection failed' > app.log

# Несколько строк
cat > app.log << 'EOF'
{"level":"info","msg":"Starting application"}
{"level":"info","msg":"Loading config from /etc/app.conf"}
{"level":"warn","msg":"Config key 'debug' not found, using default"}
{"level":"debug","msg":"Initializing database connection pool"}
{"level":"info","msg":"Connected to postgres://localhost:5432/db"}
{"level":"error","msg":"Query failed: relation 'users' does not exist"}
{"level":"info","msg":"Running migrations..."}
{"level":"info","msg":"Migration v001_create_users completed"}
{"level":"info","msg":"Server listening on :8080"}
{"level":"error","msg":"Unhandled panic: index out of range"}
{"level":"warn","msg":"Retry attempt 1/3 for external API"}
{"level":"error","msg":"External API still unavailable after 3 retries"}
EOF

# Запуск
./logt app.log

# Pipe
cat app.log | ./logt

# С другими командами
kubectl logs deployment/myapp | ./logt
journalctl -f | ./logt
docker logs -f mycontainer | ./logt
```

#### Мульти-формат (mixed sources)
```bash
# Создаём разные файлы
echo '{"level":"info","msg":"API log"}' > api.log
echo 'level=warn msg="Auth warning"' > auth.log
echo '2024-01-15 ERROR Database connection lost' > db.log

# Следим за всеми сразу
logt api.log auth.log db.log
# или
logt ./*.log
```

### Примеры
```bash
# Следить за всеми логами сервисов
logt /var/log/services/*.log

# Найти только ошибки
logt --level error ./app.log

# С настроенным буфером
logt --buffer 10000 --max-buffer 20000 ./app.log

# Экспорт отфильтрованных логов в файл
logt --forward filtered.log ./app.log

# Экспорт в stdout (pipe)
logt --forward - ./app.log | grep ERROR

# С источниками из конфига
logt

# Stdin от другой команды
kubectl logs deployment/app | logt
```

## ⌨️ Горячие клавиши

| Клавиша | Действие |
|-----|--------|
| `Space` | Пауза/Продолжить автопрокрутку |
| `/` | Открыть Fuzzy фильтр |
| `r` | Переключить Fuzzy/Regex фильтр |
| `Enter` | Применить фильтр / Открыть JSON |
| `Backspace` | Удалить символ из фильтра |
| `Esc` | Очистить фильтр / Закрыть JSON просмотр |
| `↑ / ↓` | Прокрутка вверх/вниз |
| `PgUp / PgDn` | Прокрутка по страницам |
| `Home / End` | Перейти в начало/конец |
| `g` | Перейти в начало (less-style) |
| `G` | Перейти в конец (less-style) |
| `n` | Следующее совпадение |
| `N` | Предыдущее совпадение |
| `Tab` | Переключить панель источников |
| `q` | Выход |

## 🏗️ Архитектура

```
logt/
├── cmd/logt/main.go           # Точка входа, CLI
├── internal/
│   ├── config/config.go      # Загрузка конфигурации (yaml, env)
│   ├── domain/domain.go      # Модели, Парсеры, RingBuffer
│   ├── provider/provider.go  # Провайдеры файлов и stdin
│   └── ui/
│       ├── model.go          # Состояние Bubble Tea
│       ├── update.go         # Обработчики сообщений
│       └── view.go           # Рендеринг через Lip Gloss
├── config.example.yaml       # Пример конфигурации
└── README.md
```

### Проектные решения

1. **Гексагональная архитектура** — Домен-ориентированный дизайн с чёткими границами
2. **Channel-based Concurrency** — Безопасные обновления UI через Go каналы
3. **Throttling** — UI обновляется максимум 20 раз/сек для предотвращения CPU spike
4. **Без внешней tail библиотеки** — Собственная реализация через polling

## 🧪 Тестирование

```bash
# Запустить все тесты
go test ./...

# С бенчмарками
go test -bench=. ./...

# Конкретный тест
go test -v -run TestRingBuffer ./...
```

### Покрытие тестами
- RingBuffer overflow (100k → 5k limit)
- Конкурентный доступ (thread-safety)
- JSON/Logfmt/Plain парсеры с fallback
- Fuzzy filter matching
- Определение уровня (case-insensitive)
- Конфигурация (yaml, env, flags)

## 📊 Сравнение

| Возможность | LogT | lnav |
|---------|------|------|
| Размер бинарника | ~6MB | ~15MB |
| Время старта | <100ms | ~200ms |
| JSON поддержка | Нативная | Ограниченная |
| Fuzzy Filter | ✓ | ✗ |
| Regex Filter | ✓ | ✓ |
| YAML конфиг | ✓ | ✗ |
| Требует конфиг | ✗ | ✓ |

## 🔧 Конфигурация

LogT работает **без конфига** из коробки. Для кастомизации создайте файл:

```yaml
# ~/.config/logt/config.yaml или ./logt.yaml
buffer-size: 5000     # Размер буфера
buffer-max: 10000     # Максимальный размер
theme: dark           # Тема (dark)
forward: filtered.log # Файл для экспорта
sources:              # Источники по умолчанию
  - /var/log/*.log
```

### Переменные окружения
```bash
LOGT_BUFFER_SIZE=10000
LOGT_LEVEL=error
LOGT_FORWARD=filtered.log
LOGT_THEME=dark
```

### Флаги командной строки
```
-p, --path string     Пути к файлам или шаблоны
-l, --level string   Фильтр по уровню
-b, --buffer int      Размер буфера (по умолчанию: 5000)
-m, --max-buffer int  Максимальный размер буфера
-f, --forward string  Экспорт логов (файл или stdout)
-t, --theme string    Тема (dark)
-v, --version         Версия
-h, --help            Помощь
```

## 📝 Поддерживаемые форматы логов

### Авто-определение
- **JSON** — `{"level": "error", "message": "..."}`
- **Logfmt** — `level=error msg="..."`
- **Plain** — `2024-01-01 10:00:00 ERROR message`

### Определение уровня
Без учёта регистра:
- `FATAL`, `ERROR`, `WARN`, `INFO`, `DEBUG`, `TRACE`

## 🐛 Известные ограничения

- Windows-only file watching (polling, не inotify)
- Нет удалённой поддержки логов (в планах: HTTP forwarding)
- TUI only (без headless режима)

## 🤝 Вклад

1. Fork
2. Создайте feature branch
3. Запустите тесты: `go test ./...`
4. Отправьте PR

## 📄 Лицензия

MIT License - см. [LICENSE](LICENSE)
