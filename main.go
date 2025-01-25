package main

import (
    "archive/zip"
    "bytes"
    "database/sql"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "net/http"
    "strconv"

    _ "github.com/lib/pq"
)

const (
    dbUser     = "validator"
    dbPassword = "val1dat0r"
    dbName     = "project_sem_1"
)

type Response struct {
    TotalItems     int     `json:"total_items"`
    TotalCategories int    `json:"total_categories"`
    TotalPrice     float64 `json:"total_price"`
}

func main() {
    dbInfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", dbUser, dbPassword, dbName)
    db, err := sql.Open("postgres", dbInfo)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    http.HandleFunc("/api/v0/prices", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
            return
        }
        
        var totalItems int
        categoriesSet := make(map[string]struct{})
        var totalPrice float64

        // Чтение тела запроса
        body, err := ioutil.ReadAll(r.Body)
        if err != nil {
            http.Error(w, "Unable to read request body", http.StatusBadRequest)
            return
        }

        // Использование bytes.Reader, который поддерживает io.ReaderAt
        zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
        if err != nil {
            http.Error(w, "Unable to read zip file", http.StatusBadRequest)
            return
        }

        for _, file := range zipReader.File {
            if file.Name == "data.csv" {
                f, err := file.Open()
                if err != nil {
                    http.Error(w, "Unable to open csv file", http.StatusInternalServerError)
                    return
                }
                defer f.Close()

                reader := csv.NewReader(f)
                reader.Read() // skip header

                for {
                    record, err := reader.Read()
                    if err == io.EOF {
                        break
                    }
                    if err != nil {
                        http.Error(w, "Error reading csv", http.StatusInternalServerError)
                        return
                    }

                    productID, _ := strconv.Atoi(record[0])
                    name := record[1]
                    category := record[2]
                    price, _ := strconv.ParseFloat(record[3], 64)
                    totalPrice += price
                    createDate := record[4]

                    _, err = db.Exec(`INSERT INTO prices (product_id, name, category, price, create_date) VALUES ($1, $2, $3, $4, $5)`, productID, name, category, price, createDate)
                    if err != nil {
                        http.Error(w, "Database error", http.StatusInternalServerError)
                        return
                    }

                    totalItems++
                    categoriesSet[category] = struct{}{}
                }
            }
        }

        response := Response{
            TotalItems:     totalItems,
            TotalCategories: len(categoriesSet),
            TotalPrice:     totalPrice,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    })

    log.Println("Server started at :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}