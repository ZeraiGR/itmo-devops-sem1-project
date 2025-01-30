package main

import (
	"archive/zip"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Constant values for database connection
const (
	dbUser     = "validator"
	dbPassword = "val1dat0r"
	dbName     = "project-sem-1"
	dbHost     = "localhost"
	dbPort     = "5432"
)

var database *sql.DB

// Structure for POST response
type ResponseForPost struct {
	ItemsTotal      int     `json:"total_items"`
	CategoriesTotal int     `json:"total_categories"`
	PriceTotal      float64 `json:"total_price"`
}

func initDatabase() {
	var err error
	connectionData := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)
	database, err = sql.Open("postgres", connectionData)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
}

// Обработчик для POST /api/v0/prices
func handlePostPrices(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		log.Printf("Ошибка загрузки файла: %v", err)
		http.Error(w, "Не удалось загрузить файл", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "uploaded-*.zip")
	if err != nil {
		log.Printf("Ошибка сохранения файла: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	if _, err := io.Copy(tempFile, file); err != nil {
		log.Printf("Ошибка копирования файла: %v", err)
		http.Error(w, "Ошибка копирования файла", http.StatusInternalServerError)
		return
	}

	zipReader, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		log.Printf("Ошибка открытия архива: %v", err)
		http.Error(w, "Ошибка чтения архива", http.StatusBadRequest)
		return
	}
	defer zipReader.Close()

	csvRecords, err := extractCSVRecords(&zipReader.Reader, w)

	if err != nil {
		log.Fatalf("Ошибка извлечения CSV-записей: %v", err)
	}

	tx, err := database.Begin()

	if err != nil {
		log.Printf("Ошибка начала транзакции: %v", err)
		http.Error(w, "Ошибка начала транзакции", http.StatusInternalServerError)
        tx.Rollback()
		return
	}

	if err := handleCSVRecords(tx, csvRecords); err != nil {
        tx.Rollback()
		return
	}

	response, err := calculateResponse(tx, csvRecords);

    if err != nil {
        tx.Rollback()
        return
	}

    w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
	}
}

func extractCSVRecords(zipReader *zip.Reader, w http.ResponseWriter) ([][]string, error) {
	var csvRecords [][]string

	for _, f := range zipReader.File {
		if strings.HasSuffix(f.Name, ".csv") {
			csvFile, err := f.Open()
			if err != nil {
				log.Printf("Ошибка открытия CSV: %v", err)
				http.Error(w, "Ошибка открытия CSV", http.StatusInternalServerError)
				return nil, errors.New("ошибка открытия CSV")
			}
			defer csvFile.Close()

			reader := csv.NewReader(csvFile)
			records, err := reader.ReadAll()
			if err != nil {
				log.Printf("Ошибка чтения CSV: %v", err)
				http.Error(w, "Ошибка чтения CSV", http.StatusInternalServerError)
				return nil, errors.New("ошибка чтения CSV")
			}
			csvRecords = append(csvRecords, records...)
		}
	}

	return csvRecords, nil
}

func handleCSVRecords(tx *sql.Tx, csvRecords [][]string) error {
	for _, record := range csvRecords[1:] {
		if len(record) != 5 {
			log.Printf("Invalid record length")
			continue
		}

		id := strings.TrimSpace(record[0])
		name := strings.TrimSpace(record[1])
		category := strings.TrimSpace(record[2])
		created_at := strings.TrimSpace(record[4])
		price, err := strconv.ParseFloat(strings.TrimSpace(record[3]), 64)

		if err != nil {
			log.Printf("Invalid price '%s': %v", record[3], err)
			continue
		}

		if _, err := time.Parse("2006-01-02", created_at); err != nil {
			log.Printf("Invalid create_date '%s': %v", created_at, err)
			continue
		}

		_, err = tx.Exec(`
      INSERT INTO prices (created_at, name, category, price)
      VALUES ($1, $2, $3, $4, $5)
     `, created_at, name, category, price)
		if err != nil {
			log.Printf("Database entry error for ID '%s': %v", id, err)
			continue
		}
	}
	return nil
}

func calculateResponse(tx *sql.Tx, csvRecords [][]string) (map[string]interface{}, error) {
	var totalItems int
	var totalCategories int
	var totalPrice float64

	// Подсчет общего количества элементов
	totalItems = len(csvRecords);

    // Выполнение запроса для подсчета уникальных категорий и общей суммы цен
    err := tx.QueryRow(`
    SELECT 
        COUNT(DISTINCT category) AS totalCategories, 
        COALESCE(SUM(price), 0) AS totalPrice 
    FROM prices
    `).Scan(&totalCategories, &totalPrice)

    if err != nil {
		log.Printf("Ошибка получения totalPrice, totalCategories: %v", err)
		return nil, err
	}


	// Завершаем транзакцию (коммитим все изменения)
	if err := tx.Commit(); err != nil {
		log.Printf("Ошибка подтверждения транзакции: %v", err)
		return nil, err
	}

	// Формируем и отправляем ответ
	response := map[string]interface{}{
		"total_items":      totalItems,
		"total_categories": totalCategories,
		"total_price":      totalPrice,
	}

    return response, nil;
}

func logAndRespondError(w http.ResponseWriter, logMessage string, logErr error, userMessage string, statusCode int) {
	if logErr != nil {
		log.Printf(logMessage, logErr)
	} else {
		log.Println(logMessage)
	}

	http.Error(w, userMessage, statusCode)
}

func readCSVFile(file *zip.File) ([][]string, error) {
	csvFile, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("Ошибка открытия CSV: %v", err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	return reader.ReadAll()
}

// Обработчик для GET /api/v0/prices
func handleGetPrices(w http.ResponseWriter, r *http.Request) {
	rows, err := fetchPricesFromDB()
	if err != nil {
		http.Error(w, "Ошибка чтения из базы данных", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tempDir := os.TempDir()
	csvFilePath := filepath.Join(tempDir, "data.csv")
	zipFilePath := filepath.Join(tempDir, "data.zip")

	if err := createCSVFile(rows, csvFilePath); err != nil {
		log.Println(err)
		http.Error(w, "Ошибка создания файла CSV", http.StatusInternalServerError)
		return
	}
	defer os.Remove(csvFilePath)

	if err := createZipFromFile(csvFilePath, zipFilePath); err != nil {
		log.Println(err)
		http.Error(w, "Ошибка создания архива", http.StatusInternalServerError)
		return
	}
	defer os.Remove(zipFilePath)

	serveZipFileToClient(w, r, zipFilePath)
}

func fetchPricesFromDB() (*sql.Rows, error) {
	return database.Query("SELECT id, created_at, name, category, price FROM prices")
}

func createCSVFile(rows *sql.Rows, csvFilePath string) error {
	file, err := os.Create(csvFilePath)

	if err != nil {
		return fmt.Errorf("Ошибка создания файла CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"id", "created_at", "name", "category", "price"}); err != nil {
		return err
	}

	for rows.Next() {
		var id, createdAt, name, category string
		var price float64
		if err := rows.Scan(&id, &createdAt, &name, &category, &price); err != nil {
			return fmt.Errorf("Ошибка чтения строки: %v", err)
		}
		writer.Write([]string{id, createdAt, name, category, fmt.Sprintf("%.2f", price)})
	}

    if err = rows.Err(); err != nil {
        return fmt.Errorf("Ошибка rows: %v", err)
    }

	return writer.Error()
}

func createZipFromFile(csvFilePath, zipFilePath string) error {
	archive, err := os.Create(zipFilePath)

	if err != nil {
		return fmt.Errorf("Ошибка создания архива: %v", err)
	}

	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	defer zipWriter.Close()

	csvFile, err := zipWriter.Create("data.csv")
	if err != nil {
		return fmt.Errorf("Ошибка добавления файла в архив: %v", err)
	}

	file, err := os.Open(csvFilePath)
	if err != nil {
		return fmt.Errorf("Ошибка открытия CSV файла: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(csvFile, file); err != nil {
		return fmt.Errorf("Ошибка копирования данных в архив: %v", err)
	}

	return nil
}

func serveZipFileToClient(w http.ResponseWriter, r *http.Request, zipFilePath string) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"data.zip\"")
	http.ServeFile(w, r, zipFilePath)
}

func main() {
	initDatabase()

	defer database.Close()

	http.HandleFunc("/api/v0/prices", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			handlePostPrices(w, r)
		case "GET":
			handleGetPrices(w, r)
		default:
			http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Сервер успешно запущен на порту 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
