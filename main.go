package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type RequestBodyGetServices struct {
	Action string `json:"action"`
	Key    string `json:"key"`
}

type RequestBodyGetNumber struct {
	Action   string  `json:"action"`
	Key      string  `json:"key"`
	Country  string  `json:"country"`
	Operator string  `json:"operator"`
	Service  string  `json:"service"`
	Sum      float64 `json:"sum"`
}

type reports struct {
	id     int
	url    string
	token  string
	result string
}

type get_numbers struct {
	id     int
	url    string
	token  string
	result string
}

type Country struct {
	Country     string                    `json:"country"`
	OperatorMap map[string]map[string]int `json:"operatorMap"`
}

type Response struct {
	CountryList []Country `json:"countryList"`
	Status      string    `json:"status"`
}

func init() {
	var err error
	db, err = sql.Open("sqlite3", "mydatabase.db")
	if err != nil {
		log.Fatal(err)
	}
}

func createReportsTable(db *sql.DB) error {
	// Удаляем таблицу, если она существует
	_, err := db.Exec("DROP TABLE IF EXISTS reports")
	if err != nil {
		return err
	}

	// Удаляем таблицу, если она существует
	_, err1 := db.Exec("DROP TABLE IF EXISTS get_numbers")
	if err1 != nil {
		return err1
	}

	// Создаем таблицу
	if _, err = db.Exec(`
        CREATE TABLE reports (
            id TEXT PRIMARY KEY,
            url TEXT,
            token TEXT,
            result TEXT,
            status TEXT,
            stage TEXT
        );
    `); err != nil {
		return err
	}

	// Создаем таблицу get_numbers
	if _, err = db.Exec(`
        CREATE TABLE get_numbers (
            id TEXT PRIMARY KEY,
            url TEXT,
            token TEXT,
            result TEXT,
            status TEXT,
            stage TEXT
        );
    `); err != nil {
		return err
	}

	return nil
}

func handleForm(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		html := `
            <html>
                <body>
                    <form action="/" method="post">
                        <label for="url">URL:</label>
                        <input type="text" id="url" name="url"><br><br>
                        <label for="token">Token:</label>
                        <input type="text" id="token" name="token"><br><br>
                        <input type="submit" value="Отправить">
                    </form>
                </body>
            </html>
        `
		fmt.Fprint(w, html)
		return
	}

	if r.Method == "POST" {
		url := r.FormValue("url")
		token := r.FormValue("token")

		log.Println("URL:", url)
		log.Println("Token:", token)

		// Запуск тестирования
		reportID := createReport(url, token)

		// Перенаправляет на ссылку с отчетом
		http.Redirect(w, r, fmt.Sprintf("/report/%s", reportID), http.StatusFound)
		return
	}

	http.Error(w, "Invalid request method", http.StatusBadRequest)
}

func testProtocol_GET_NUMBER(url string, token string, country string, service string) string {
	// Реализация логики GET_NUMBER
	requestBodyGetNumber := RequestBodyGetNumber{
		Action:   "GET_NUMBER",
		Key:      token,
		Country:  country,
		Operator: "any",
		Service:  service,
		Sum:      20.00,
	}

	// Конвертируем структуру в JSON
	jsonBodyGetNumber, err := json.Marshal(requestBodyGetNumber)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	// Создаем HTTP-запрос для GET_NUMBER
	reqGetNumber, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBodyGetNumber))
	if err != nil {
		fmt.Println(err)
		return ""
	}

	// Устанавливаем заголовки
	reqGetNumber.Header.Set("User-Agent", "MyClient/1.0 (Go)")
	reqGetNumber.Header.Set("Content-Type", "application/json")

	// Отправляем запрос для GET_NUMBER
	respGetNumber, err := http.DefaultClient.Do(reqGetNumber)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer respGetNumber.Body.Close()

	// Проверяем статус ответа для GET_NUMBER
	if respGetNumber.StatusCode != http.StatusOK {
		fmt.Println("Failed to get number")
		return ""
	}

	// Печатаем ответ
	fmt.Println("Status:", respGetNumber.Status)
	fmt.Println("Headers:")
	for name, values := range respGetNumber.Header {
		fmt.Println(name, ":", values)
	}
	fmt.Println("Body:")
	body, err := ioutil.ReadAll(respGetNumber.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	fmt.Println(string(body))

	return string(body)
}

func testProtocol_GET_SERVICES(url string, token string) (string, Response, error) {
	// Реализация логики GET_SERVICES
	requestBody := RequestBodyGetServices{
		Action: "GET_SERVICES",
		Key:    token,
	}

	// Конвертируем структуру в JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", Response{}, err
	}

	// Создаем HTTP-запрос
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", Response{}, err
	}

	// Устанавливаем заголовки
	req.Header.Set("User-Agent", "MyClient/1.0 (Go)")
	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", Response{}, err
	}

	// Печатаем ответ
	defer resp.Body.Close()
	fmt.Println("Status:", resp.Status)
	fmt.Println("Headers:")
	for name, values := range resp.Header {
		fmt.Println(name, ":", values)
	}
	fmt.Println("Body:")

	// Unmarshal ответа в структуру
	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", Response{}, err
	}

	// Конвертируем ответ в строку
	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", Response{}, err
	}

	return string(responseJSON), response, nil
}

func createReport(url string, token string) string {
	// Создаем таблицы, если они еще не существует
	if err := createReportsTable(db); err != nil {
		log.Fatal(err)
	}

	// Генерация уникального идентификатора отчета
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		log.Fatal(err)
	}
	reportID := hex.EncodeToString(bytes)

	responseJSON, response, err := testProtocol_GET_SERVICES(url, token)
	if err != nil {
		fmt.Println("Ошибка:", err)
		return ""
	}

	fmt.Println("Ответ в виде структуры:")
	country := response.CountryList[0].Country
	fmt.Println(country)
	fmt.Println("OperatorMap:")
	fmt.Println(response.CountryList[0].OperatorMap)

	firstKey := ""
	firstValue := make(map[string]int)
	for key, value := range response.CountryList[0].OperatorMap {
		firstKey = key
		firstValue = value
		fmt.Printf("Ключ первого уровня: %s\n", firstKey)
		break
	}
	var resultGetNumber string
	var count string
	var redactString string
	for k, v := range firstValue {
		fmt.Printf("  Ключ второго уровня: %s, Значение: %d\n", k, v)
		count = strconv.Itoa(v)
		resultGetNumber := testProtocol_GET_NUMBER(url, token, country, k)
		print("    ответ каждой записи:", resultGetNumber)
		redactString = "заявлено " + count + ", получено " + resultGetNumber
		break
	}

	// Проверяем количество номеров
	if count != resultGetNumber {
		// Сохраняем результаты в базе данных с статусом error
		_, err = db.Exec(`
			INSERT INTO get_numbers (id, url, token, result, status, stage)
			VALUES (?, ?, ?, ?, ?, ?);
		`, reportID, url, token, redactString, "error", "done")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// Сохраняем результаты в базе данных с статусом success
		_, err = db.Exec(`
			INSERT INTO get_numbers (id, url, token, result, status, stage)
			VALUES (?, ?, ?, ?, ?, ?);
		`, reportID, url, token, redactString, "success", "done")
		if err != nil {
			log.Fatal(err)
		}
	}

	// Сохраняем результаты в базе данных
	_, err = db.Exec(`
		INSERT INTO reports (id, url, token, result, status, stage)
		VALUES (?, ?, ?, ?, ?, ?);
	`, reportID, url, token, responseJSON, "success", "done")
	if err != nil {
		log.Fatal(err)
	}

	return reportID
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	reportID := r.URL.Path[len("/report/"):]
	row := db.QueryRow(`
        SELECT result, status, stage
        FROM reports
        WHERE id = ?;
    `, reportID)

	var result, status, stage string
	err := row.Scan(&result, &status, &stage)
	if err != nil {
		http.Error(w, "Отчет не найден", http.StatusNotFound)
		return
	}

	// Еще один запрос для GET_NUMBER
	row2 := db.QueryRow(`
        SELECT result, status, stage
        FROM get_numbers
        WHERE id = ?;
    `, reportID)

	var result2, status2, stage2 string
	err = row2.Scan(&result2, &status2, &stage2)
	if err != nil {
		http.Error(w, "Второй отчет не найден", http.StatusNotFound)
		return
	}

	html := fmt.Sprintf(`
        <html>
            <body>
                <h1>Отчет о тестировании GET_SERVICES</h1>
                <p>Статус: %s</p>
                <p>Этап: %s</p>
                <p>Результат: %s</p>
                <h1>Отчет о тестировании GET_NUMBER</h1>
                <p>Статус: %s</p>
                <p>Этап: %s</p>
                <p>Результат: %s</p>
            </body>
        </html>
    `, status, stage, result, status2, stage2, result2)

	fmt.Fprint(w, html)
}

func main() {
	// logic service
	http.HandleFunc("/", handleForm)
	http.HandleFunc("/report/", reportHandler)
	http.ListenAndServe(":8080", nil)
}
