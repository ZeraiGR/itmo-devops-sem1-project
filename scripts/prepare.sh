#!/bin/bash

set -e

# Переменные для подключения
PGHOST="localhost"
PGPORT=5432
PGUSER="validator"
PGPASSWORD="val1dat0r"
DBNAME="project-sem-1"
# LOCALPGUSER="kirill-gruzdy"
# LOCALDBNAME="postgres"

export PGPASSWORD

unameOut="$(uname -s)"
case "${unameOut}" in
    Linux*)     machine=Linux;;
    Darwin*)    machine=Mac;;
    CYGWIN*)    machine=Cygwin;;
    MINGW*)     machine=MinGw;;
    *)          machine="UNKNOWN:${unameOut}"
esac

echo ${machine}

if [ "$machine" == "Mac" ]; then
    echo "run mac os specific flow"
    psql -d ${DBNAME} -U ${PGUSER} <<-EOSQL
    DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_user WHERE usename = 'validator') THEN
        CREATE USER validator WITH PASSWORD 'val1dat0r';
    END IF;
    END \$\$;

    DO \$\$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_database WHERE datname = '${DBNAME}') THEN
        CREATE DATABASE "${DBNAME}" OWNER validator;
    END IF;
    END \$\$;

    GRANT ALL PRIVILEGES ON DATABASE "${DBNAME}" TO validator;
EOSQL
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then

    echo "Проверяем доступность базы данных..."
    if ! psql -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" -d "$DBNAME" -c "\\q" &> /dev/null; then
    echo "База данных $DBNAME недоступна. Проверяем настройки..."
    
    echo "Пробуем подключиться как postgres..."
    PGUSER="postgres"
    if ! psql -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" -c "\\q" &> /dev/null; then
        echo "Ошибка: Не удалось подключиться к базе данных как postgres."
        exit 1
    fi

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

        GRANT ALL PRIVILEGES ON DATABASE "${DBNAME}" TO validator;
EOSQL

    else
        echo "База данных $DBNAME доступна. Ничего не требуется."
    fi

fi

echo "Создаем таблицу prices..."
PGUSER="validator"
psql -U "$PGUSER" -h "$PGHOST" -p "$PGPORT" -d "$DBNAME" <<-EOSQL
  DROP TABLE prices;
  CREATE TABLE IF NOT EXISTS prices (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    created_at DATE NOT NULL
  );
EOSQL

echo "Подготовка базы данных успешно завершена."