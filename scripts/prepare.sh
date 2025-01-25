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

echo "PostgreSQL успешно установлен и запущен."

echo "Подготовка базы данных..."

# Данные для подключения к базе данных администратора
DB_ADMIN="postgres"
DB_NAME="project_sem_1"
DB_USER="validator"
DB_PASSWORD="val1dat0r"

# Создание базы данных и пользователя
echo "Создание базы данных $DB_NAME и пользователя $DB_USER..."
sudo -u $DB_ADMIN psql postgres <<EOSQL
CREATE USER $DB_USER WITH ENCRYPTED PASSWORD '$DB_PASSWORD';
CREATE DATABASE $DB_NAME;
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;
\c $DB_NAME
CREATE TABLE prices (
    product_id INTEGER PRIMARY KEY,
    name TEXT,
    category TEXT,
    price NUMERIC,
    create_date TIMESTAMP
);
EOSQL

echo "База данных подготовлена."

echo "Скрипт подготовки завершен."