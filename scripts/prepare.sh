#!/bin/bash

# Завершаем выполнение при ошибках
set -e

echo "Создание базы данных"

# Переменные для подключения
PGHOST="localhost"
PGPORT=5432
PGUSER="validator"
PGPASSWORD="val1dat0r"
DBNAME="project-sem-1"

export PGPASSWORD

# Проверка доступности базы данных
echo "Проверка доступности базы данных"
if ! psql -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" -d "$DBNAME" -c "\\q" &> /dev/null; then
  echo "База данных $DBNAME недоступна. Проверяем настройки."
  
  # Проверка подключения с пользователем postgres
  echo "Подключение под postgres"
  PGUSER="postgres"
  if ! psql -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" -c "\\q" &> /dev/null; then
    echo "Ошибка: Не удалось подключиться как postgres."
    exit 1
  fi

  # Создание пользователя и базы данных
  echo "Создаём пользователя и базу данных..."
  psql -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" <<-EOSQL
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
else
  echo "База данных $DBNAME доступна."
fi

echo "Подготовка базы данных завершена успешно."