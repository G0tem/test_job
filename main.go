package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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

type GetNumberResponse struct {
	ActivationID int    `json:"activationID"`
	Number       string `json:"number"`
	Status       string `json:"status"`
}

func init() {
	var err error
	db, err = sql.Open("sqlite3", "mydatabase.db")
	if err != nil {
		log.Fatal(err)
	}
}

func createReportsTable(db *sql.DB) error {
	// Удаляем таблицы, если она существует
	_, err := db.Exec("DROP TABLE IF EXISTS reports")
	if err != nil {
		return err
	}

	_, err1 := db.Exec("DROP TABLE IF EXISTS get_numbers")
	if err1 != nil {
		return err1
	}

	// Создаем таблицы
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

func testProtocol_GET_NUMBER(url string, token string, country string, service string) (string, GetNumberResponse, error) {
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
		return "", GetNumberResponse{}, err
	}

	// Создаем HTTP-запрос для GET_NUMBER
	reqGetNumber, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBodyGetNumber))
	if err != nil {
		return "", GetNumberResponse{}, err
	}

	// Устанавливаем заголовки
	reqGetNumber.Header.Set("User-Agent", "MyClient/1.0 (Go)")
	reqGetNumber.Header.Set("Content-Type", "application/json")

	// Отправляем запрос для GET_NUMBER
	respGetNumber, err := http.DefaultClient.Do(reqGetNumber)
	if err != nil {
		return "", GetNumberResponse{}, err
	}
	defer respGetNumber.Body.Close()

	// Проверяем статус ответа для GET_NUMBER
	if respGetNumber.StatusCode != http.StatusOK {
		return "", GetNumberResponse{}, errors.New("Failed to get number")
	}

	// Читаем тело ответа
	body, err := ioutil.ReadAll(respGetNumber.Body)
	if err != nil {
		return "", GetNumberResponse{}, err
	}

	// Десериализуем ответ в структуру
	var response GetNumberResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", GetNumberResponse{}, err
	}

	// Возвращаем ответ в виде строки и в виде структуры
	return string(body), response, nil
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

	country := response.CountryList[0].Country
	var countNumber int
	var service string
	for _, value := range response.CountryList[0].OperatorMap {
		for k, v := range value {
			countNumber = v
			service = k
			break
		}
		break
	}

	numbers, err := getNumbers(url, token, country, countNumber, service)
	if err != nil {
		log.Fatal(err)
	}

	message_info := fmt.Sprintf("заявлено " + strconv.Itoa(countNumber) + ", получено " + strconv.Itoa(len(numbers)))

	// Проверяем количество номеров
	if len(numbers) != countNumber {
		// Сохраняем результаты в базе данных с статусом error
		_, err = db.Exec(`
            INSERT INTO get_numbers (id, url, token, result, status, stage)
            VALUES (?, ?, ?, ?, ?, ?);
        `, reportID, url, token, message_info, "error", "done")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// Сохраняем результаты в базе данных с статусом success
		_, err = db.Exec(`
            INSERT INTO get_numbers (id, url, token, result, status, stage)
            VALUES (?, ?, ?, ?, ?, ?);
        `, reportID, url, token, message_info, "success", "done")
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

func getNumbers(url string, token string, country string, countNumber int, service string) ([]string, error) {
	// Создаем map для хранения уникальных номеров
	numbersMap := make(map[string]bool)

	// Создаем слайс для хранения уникальных номеров
	var numbers []string

	// Канал для хранения результатов запросов
	results := make(chan string)

	// Запускаем горутину для каждого запроса
	for i := 0; i < countNumber; i++ {
		go func() {
			_, getNumberResponse, err := testProtocol_GET_NUMBER(url, token, country, service)
			if err != nil {
				fmt.Println("Ошибка:", err)
				return
			}
			results <- getNumberResponse.Number
		}()
	}

	// Ожидаем результаты запросов
	for i := 0; i < countNumber; i++ {
		number := <-results
		if _, ok := numbersMap[number]; !ok {
			numbersMap[number] = true
			numbers = append(numbers, number)
		}
	}
	return numbers, nil
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

	// Запрос для GET_NUMBER
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
	http.HandleFunc("/", handleForm)
	http.HandleFunc("/report/", reportHandler)
	http.ListenAndServe(":8080", nil)
}
