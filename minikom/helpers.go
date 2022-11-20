package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/golang/gddo/httputil/header"
)

type malformedRequest struct {
	status int
	msg    string
}

type serviceSeen struct {
	State string `json:"state"`
	Since int64  `json:"since"`
}

type serviceState struct {
	State string
	Dt    string
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

type Postgres struct {
	Db  *sql.DB
	cfg Config
}

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

func New(cfg Config) (postgres Postgres, err error) {
	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" ||
		cfg.Password == "" || cfg.Database == "" {
		err = errors.Errorf(
			"All fields must be set (%s)",
			spew.Sdump(cfg))
		return
	}

	postgres.cfg = cfg

	db, err := sql.Open("postgres", fmt.Sprintf(
		"user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		cfg.User, cfg.Password, cfg.Database, cfg.Host, cfg.Port))
	if err != nil {
		err = errors.Wrapf(err,
			"Couldn't open connection to postgre database (%s)",
			spew.Sdump(cfg))
		return
	}

	if err = db.Ping(); err != nil {
		err = errors.Wrapf(err,
			"Couldn't ping postgres database (%s)",
			spew.Sdump(cfg))
		return
	}

	postgres.Db = db
	return
}

func (r *Postgres) Close() (err error) {
	if r.Db == nil {
		return
	}

	if err = r.Db.Close(); err != nil {
		err = errors.Wrapf(err,
			"Errored closing database connection",
			spew.Sdump(r.cfg))
	}

	return
}

func (mr *malformedRequest) Error() string {
	return mr.msg
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	if r.Header.Get("Content-Type") != "" {
		value, _ := header.ParseValueAndParams(r.Header, "Content-Type")
		if value != "application/json" {
			msg := "Content-Type header is not application/json"
			return &malformedRequest{status: http.StatusUnsupportedMediaType, msg: msg}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(&dst)

	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.ErrUnexpectedEOF):
			msg := fmt.Sprintln("Request body contains badly-formed JSON")
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			return &malformedRequest{status: http.StatusBadRequest, msg: msg}

		case err.Error() == "http: request body too large":
			msg := "Request body must not be larger than 1MB"
			return &malformedRequest{status: http.StatusRequestEntityTooLarge, msg: msg}

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		msg := "Request body must only contain a single JSON object"
		return &malformedRequest{status: http.StatusBadRequest, msg: msg}
	}

	return nil
}

func getLatestServices() map[string]serviceSeen {
	var service_name string
	var state string
	var dt string
	var result = map[string]serviceSeen{}

	statement := `
		SELECT DISTINCT ON (service_name)
			service_name, state, dt
		FROM services
		ORDER BY service_name, dt DESC;`
	rows, err := postgres.Db.Query(statement)
	checkError(err)
	defer rows.Close()

	for rows.Next() {
		switch err := rows.Scan(&service_name, &state, &dt); err {
		case sql.ErrNoRows:
			fmt.Println("No rows were returned")
		case nil:
			var r serviceSeen
			timestamp, _ := time.Parse(time.RFC3339, dt)

			r.State = state
			r.Since = timestamp.Unix()
			result[service_name] = r
		default:
			checkError(err)
		}
	}

	return result
}

func getServiceStatesByName(service_name string) []serviceState {
	var state string
	var dt string
	var states []serviceState

	statement := `
		SELECT state, dt
		FROM services
		WHERE service_name = $1
		ORDER BY dt;`
	rows, err := postgres.Db.Query(statement, service_name)
	checkError(err)

	for rows.Next() {
		switch err := rows.Scan(&state, &dt); err {
		case sql.ErrNoRows:
			fmt.Println("No rows were returned")
		case nil:
			states = append(states, serviceState{State: state, Dt: dt})
		default:
			checkError(err)
		}
	}

	return states
}

func getLatestService(service_name string) string {
	state := ""
	statement := `
		SELECT state
		FROM services
		WHERE service_name=$1
		ORDER BY dt DESC LIMIT 1`
	row := postgres.Db.QueryRow(statement, service_name)

	switch err := row.Scan(&state); err {
	case sql.ErrNoRows:
		fmt.Println("No rows were returned!")
	case nil:
		return state
	default:
		panic(err)
	}

	return state
}

func saveService(service_name string, state string, timestamp time.Time) {
	statement := `
		INSERT INTO services (service_name, state, dt)
		VALUES ($1, $2, $3)`
	_, err := postgres.Db.Exec(statement, service_name, state, timestamp)
	checkError(err)
}

func saveEvent(service_name string, state string, timestamp time.Time) {
	statement := `
		INSERT INTO events (service_name, state, dt)
		VALUES ($1, $2, $3)`
	_, err := postgres.Db.Exec(statement, service_name, state, timestamp)
	checkError(err)
}
