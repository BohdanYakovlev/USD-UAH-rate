package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"io"
	"log"
	"net/http"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type Rate struct {
	Ccy     string `json:"ccy"`
	BaseCcy string `json:"base_ccy"`
	Buy     string `json:"buy"`
	Sale    string `json:"sale"`
}

var db *sql.DB

const (
	host     = "postgres"
	port     = 5432
	user     = "techuser"
	password = "techuser"
	dbname   = "postgres"
)
const emailInDBStatusCode = 409
const rateUrl = "https://api.privatbank.ua/p24api/pubinfo?json&exchange&coursid=5"

var psqlInfo string

func connectDB() {
	var err error

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}

func inDB(email string) bool {
	row := db.QueryRow(sqlSelect, email)

	var temp string

	switch err := row.Scan(&temp); err {
	case sql.ErrNoRows:
		return false
	default:
		log.Fatal(err)
	}
	return true
}

func getRate(w http.ResponseWriter, r *http.Request) {
	var client http.Client
	resp, err := client.Get(rateUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		rates := []Rate{}

		err = json.Unmarshal(bodyBytes, &rates)
		responseBytes, _ := json.Marshal(rates[1].Buy)
		w.WriteHeader(200)
		w.Write(responseBytes)
		return
	}
}

func setSubscribeEmail(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	email := params["email"]

	if inDB(email) {
		w.WriteHeader(emailInDBStatusCode)
		return
	}

	_, err := db.Exec(sqlInsert, email)
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(200)
}

func getEmails(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(selectAll)
	if err != nil {
		log.Fatal(err)
	}

	var result []string

	for rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			log.Fatal(err)
		}
		result = append(result, email)
	}

	emailsBytes, _ := json.Marshal(result)
	w.WriteHeader(200)
	w.Write(emailsBytes)
}

func migrateDB() {
	m, err := migrate.New(
		"./db/migrations", // FAILS WITH ERROR "Failed to create migrate instance: no scheme"
		psqlInfo,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to apply migrations: %v", err)
	}
}

func main() {
	psqlInfo = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	connectDB()
	defer db.Close()

	log.Println("Starting init DB SQL")
	//migrateDB()
	_, err := db.Exec(initSQL)
	if err != nil {
		log.Printf("Error during init DB SQL: %s", err)
		return
	}
	log.Println("Done init DB SQL")

	router := mux.NewRouter()

	router.HandleFunc("/api/rate", getRate).Methods("GET")
	router.HandleFunc("/api/emails", getEmails).Methods("GET")
	router.HandleFunc("/api/subscribe/{email}", setSubscribeEmail).Methods("POST")
	http.ListenAndServe(":8001", router)
}

const sqlInsert = `
	INSERT INTO users (email)
	VALUES ($1)
	RETURNING id`

const sqlSelect = `
	SELECT id
	FROM users
	WHERE email=$1`

const selectAll = `
	SELECT email
	FROM users`

const initSQL = `
	CREATE TABLE users(
	id INTEGER PRIMARY KEY,
	email VARCHAR(100) NOT NULL UNIQUE
	)
`
