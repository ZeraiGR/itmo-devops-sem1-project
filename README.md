# Финальный проект 1 семестра

REST API сервис для загрузки и выгрузки данных о ценах.

## Требования к системе

Установлен PostgreSQL не менее 14 версии.

## Установка и запуск

Установите PostgreSQL на машинку:

for linux
```
sudo apt update
sudo apt install -y postgresql
```

for mac os
```
brew update
brew install postgresql
```

Подготовьте БД, запустив скрипт:
`./scripts/prepare.sh`

Запустите сервер:
`./scripts/run.sh`

## Тестирование

Директория `sample_data` - это пример директории, которая является разархивированной версией файла `sample_data.zip`

`curl -X POST -F "file=@./sample_data.zip" http://localhost:8080/api/v0/prices`

## Контакт

К кому можно обращаться в случае вопросов? https://github.com/ZeraiGR
