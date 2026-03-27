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
```

### Переменные окружения

```bash
export LOGT_BUFFER_SIZE=10000
export LOGT_LEVEL=error
export LOGT_FORWARD=out.log
```

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
