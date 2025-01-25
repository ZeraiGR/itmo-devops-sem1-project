package main

import (
 "archive/zip"
 "database/sql"
 "encoding/csv"
 "encoding/json"
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
    ItemsTotal    int     `json:"total_items"`
    CategoriesTotal int   `json:"total_categories"`
    PriceTotal    float64 `json:"total_price"`
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
    file, err := getFileFromRequest(r)
    if err != nil {
     logAndRespondError(w, "Ошибка загрузки файла: %v", err, "Не удалось загрузить файл", http.StatusBadRequest)
     return
    }

    defer file.Close()
   
    tempFile, err := saveFileToTemp(file)
    if err != nil {
     logAndRespondError(w, "Ошибка сохранения файла: %v", err, "Ошибка сохранения файла", http.StatusInternalServerError)
     return
    }
    defer os.Remove(tempFile.Name())
   
    csvRecords, err := extractCSVRecordsFromZip(tempFile.Name())
    if err != nil {
     logAndRespondError(w, err.Error(), nil, err.Error(), http.StatusInternalServerError)
     return
    }
   
    if err := processCSVRecords(csvRecords, w); err != nil {
     log.Println(err)
     http.Error(w, "Не удалось обработать CSV записи", http.StatusInternalServerError)
    }
}

func logAndRespondError(w http.ResponseWriter, logMessage string, logErr error, userMessage string, statusCode int) {
	if logErr != nil {
	 	log.Printf(logMessage, logErr)
	} else {
	 	log.Println(logMessage)
	}

	http.Error(w, userMessage, statusCode)
}

func getFileFromRequest(r *http.Request) (multipart.File, error) {
    file, _, err := r.FormFile("file")
    return file, err
}

func saveFileToTemp(file multipart.File) (*os.File, error) {
    tempFile, err := os.CreateTemp("", "uploaded-*.zip")
    if err != nil {
     return nil, err
    }
   
    if _, err = io.Copy(tempFile, file); err != nil {
     return nil, err
    }
   
    if err = tempFile.Close(); err != nil {
     return nil, err
    }
   
    return tempFile, nil
}

func extractCSVRecordsFromZip(fileName string) ([][]string, error) {
    zipReader, err := zip.OpenReader(fileName)
    if err != nil {
     return nil, fmt.Errorf("Ошибка открытия архива: %v", err)
    }
    defer zipReader.Close()
   
    var csvRecords [][]string
    for _, f := range zipReader.File {
     if !strings.HasSuffix(f.Name, ".csv") {
      continue
     }
   
     records, err := readCSVFile(f)
     if err != nil {
      return nil, err
     }
     csvRecords = append(csvRecords, records...)
    }
    return csvRecords, nil
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

func processCSVRecords(csvRecords [][]string, w http.ResponseWriter) error {
    tx, err := database.Begin()
    if err != nil {
     return fmt.Errorf("Ошибка при попытке начать транзакцию: %v", err)
    }
    defer tx.Rollback()
   
    for _, record := range csvRecords {
     if err := processRecord(tx, record); err != nil {
      log.Println(err)
      continue
     }
    }
   
    return calculateAndRespondTotals(tx, w)
}

func processRecord(tx *sql.Tx, record []string) error {
    if len(record) < 5 {
     return nil
    }
   
    id, createdAt, name, category, price, err := parseRecord(record)
    if err != nil {
     return err
    }
   
    _, err = tx.Exec(`
     INSERT INTO prices (id, created_at, name, category, price)
     VALUES ($1, $2, $3, $4, $5)
    `, id, createdAt, name, category, price)
    if err != nil {
     return fmt.Errorf("Ошибка во время записи в базу данных для ID '%s': %v", id, err)
    }
    return nil
}

func parseRecord(record []string) (string, string, string, string, float64, error) {
    id := strings.TrimSpace(record[0])
    createdAt := strings.TrimSpace(record[1])
    name := strings.TrimSpace(record[2])
    category := strings.TrimSpace(record[3])
    price, err := strconv.ParseFloat(strings.TrimSpace(record[4]), 64)
    if err != nil {
     return "", "", "", "", 0, fmt.Errorf("Ошибка при преобразовании цены '%s': %v", record[4], err)
    }
   
    if _, err = time.Parse("2006-01-02", createdAt); err != nil {
     return "", "", "", "", 0, fmt.Errorf("Некорректный формат даты '%s': %v", createdAt, err)
    }
    return id, createdAt, name, category, price, nil
}

func calculateAndRespondTotals(tx *sql.Tx, w http.ResponseWriter) error {
    totalItems, totalCategories, totalPrice, err := calculateTotals(tx)

    if err != nil {
        return err
    }
   
    if err = tx.Commit(); err != nil {
        return fmt.Errorf("Ошибка подтверждения транзакции: %v", err)
    }
   
    response := ResponseForPost{
        ItemsTotal:     totalItems,
        CategoriesTotal: totalCategories,
        PriceTotal:     totalPrice,
    }

    w.Header().Set("Content-Type", "application/json")
    return json.NewEncoder(w).Encode(response)
}

func calculateTotals(tx *sql.Tx) (int, int, float64, error) {
	var totalItems int
	var totalCategories int
	var totalPrice float64
   
	err := tx.QueryRow("SELECT COUNT(*) FROM prices").Scan(&totalItems)
	if err != nil {
	 return 0, 0, 0, fmt.Errorf("Ошибка получения total_items: %v", err)
	}
   
	err = tx.QueryRow("SELECT COUNT(DISTINCT category) FROM prices").Scan(&totalCategories)
	if err != nil {
	 return 0, 0, 0, fmt.Errorf("Ошибка получения total_categories: %v", err)
	}
   
	err = tx.QueryRow("SELECT SUM(price) FROM prices").Scan(&totalPrice)
	if err != nil {
	 return 0, 0, 0, fmt.Errorf("Ошибка получения total_price: %v", err)
	}
   
	return totalItems, totalCategories, totalPrice, nil
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