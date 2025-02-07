# Используем официальный образ Go для сборки
FROM golang:1.23-alpine as builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы проекта в контейнер
COPY . .

# Загружаем зависимости и собираем бинарный файл
RUN go mod tidy && go build -o pastebin

# Используем минимальный образ для финального контейнера
FROM alpine:latest

# Устанавливаем рабочую директорию
WORKDIR /root/

# Копируем бинарный файл из контейнера сборки
COPY --from=builder /app/pastebin .
COPY --from=builder /app/.env .
COPY --from=builder /app/web/ ./web
COPY --from=builder /app/logs/ ./logs

# Создаем папку для логов, если её нет
RUN mkdir -p /root/logs && touch /root/logs/paste_actions.log


# Указываем команду запуска
CMD ["./pastebin"]
