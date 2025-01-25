#!/bin/bash

set -e

echo "Начало установки зависимостей приложения..."

# Установка зависимости для Go проекта
go get -u github.com/lib/pq

echo "Зависимости проекта установлены."

echo "Обновление списка пакетов..."
sudo apt update

echo "Установка PostgreSQL..."
sudo apt install -y postgresql postgresql-contrib

echo "Запуск службы PostgreSQL..."
sudo systemctl start postgresql
sudo systemctl enable postgresql

echo "Проверка состояния службы PostgreSQL и ожидание её готовности..."
# Ждем пока сокет сервера не станет доступен
until pg_isready -h localhost
do
  echo "Ожидание запуска PostgreSQL..."
  sleep 1
done

echo "PostgreSQL успешно установлен и запущен."

echo "Подготовка базы данных..."

# Данные для подключения к базе данных администратора
DB_HOST="localhost"
DB_PORT=5432
DB_ADMIN="postgres"
DB_NAME="project-sem-1"
DB_USER="validator"
DB_PASSWORD="val1dat0r"

# Создание базы данных и пользователя
echo "Создание базы данных $DB_NAME и пользователя $DB_USER..."
psql -U "$DB_USER" -h "$DB_HOST" -p "$DB_PORT" <<EOSQL
DO \$\$ BEGIN
      IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = 'validator') THEN
        CREATE USER validator WITH PASSWORD 'val1dat0r';
      END IF;
    END \$\$;

    DO \$\$ BEGIN
      IF NOT EXISTS (SELECT FROM pg_database WHERE datname = '${DBNAME}') THEN
        CREATE DATABASE ${DBNAME} OWNER validator;
      END IF;
    END \$\$;

    GRANT ALL PRIVILEGES ON DATABASE ${DBNAME} TO validator;
EOSQL

echo "База данных подготовлена."

echo "Скрипт подготовки завершен."