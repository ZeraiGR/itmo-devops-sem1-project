#!/bin/bash

# Проверяем, установлен ли make
if ! command -v make &> /dev/null; then
  echo "make не установлен. Устанавливаем..."

  # Определяем ОС и устанавливаем make
  if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Для Linux (Debian/Ubuntu)
    sudo apt-get update
    sudo apt-get install -y make
  elif [[ "$OSTYPE" == "darwin"* ]]; then
    # Для macOS
    brew install make
  else
    echo "Ошибка: Неподдерживаемая операционная система."
    exit 1
  fi

  echo "make успешно установлен."
else
  echo "make уже установлен."
fi

# Выполняем make setup
echo "Запуск make setup..."
if ! make setup; then
  echo "Ошибка: make setup завершился с ошибкой."
  exit 1
fi

echo "Обновление зависимостей"
go mod tidy

# Выполняем make lint/sources и проверяем вывод на ошибки
echo "Запуск make lint/sources..."
if ! make lint/sources; then
  echo "Ошибка: make lint/sources завершился с ошибкой."
  exit 1
fi

echo "Сборка приложения"
go build -o app cmd/main.go

echo "Запуск приложения"
./app &

# Сохраняем PID запущенного процесса
APP_PID=$!
echo "Приложение запущено с PID: $APP_PID"

# Ожидание доступности сервера
echo "Ожидаем, пока сервер будет готов..."
for i in {1..10}; do
  if curl -s http://localhost:8080 &> /dev/null; then
    echo "Сервер успешно запущен и готов к работе."
    exit 0
  fi
  echo "Попытка $i: сервер пока не готов..."
  sleep 2
done

echo "Ошибка: сервер не запустился."
exit 1