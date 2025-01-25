#!/bin/bash

echo "Стартуем приложение..."
go run main.go &

# Ожидание доступности сервера
echo "Ждем, пока сервер не станет доступным..."
for attempt in {1..20}; do
  if curl -s http://localhost:8080 &> /dev/null; then
    echo "Сервер успешно запущен и готов к использованию."
    exit 0
  fi
  echo "Попытка номер $attempt: сервер пока недоступен..."
  sleep 2
done

echo "Сервер не сумел запуститься в течение ожидаемого времени."
exit 1