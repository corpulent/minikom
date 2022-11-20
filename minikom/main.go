package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var postgres Postgres

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Dbname   string
}

type Event struct {
	Service_Name string
	State        string
	TimeStamp    int64
}

type serviceStateJson struct {
	State      string `json:"state"`
	Start_Time int64  `json:"start_time"`
	End_Time   int64  `json:"end_time"`
}

const (
	OK      = "OK"
	BAD     = "BAD"
	TIMEOUT = "TIMEOUT"
)

type State struct {
	status string
}

func NewState() *State {
	return &State{status: OK}
}

func (s *State) Health(rw http.ResponseWriter, r *http.Request) {
	log.Printf("Received /health request: source=%v status=%v", r.RemoteAddr, s.status)
	switch s.status {
	case OK:
		io.WriteString(rw, "I'm healthy")
	case BAD:
		http.Error(rw, "Internal Error", 500)
	case TIMEOUT:
		time.Sleep(30 * time.Second)
	default:
		io.WriteString(rw, "UNKNOWN")
	}
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var e Event

		err := decodeJSONBody(w, r, &e)

		if err != nil {
			var mr *malformedRequest
			if errors.As(err, &mr) {
				http.Error(w, mr.msg, mr.status)
				fmt.Printf("Error: %+v \n", mr.msg)
			} else {
				fmt.Printf("Error: %+v \n", err.Error())
				log.Print(err.Error())
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}

		tm := time.Unix(e.TimeStamp, 0)
		latestState := getLatestService(e.Service_Name)

		if latestState != e.State {
			saveService(e.Service_Name, e.State, tm)
		}

		fmt.Printf("new event %s %s at %s \n", e.Service_Name, e.State, tm)

		saveEvent(e.Service_Name, e.State, tm)
	default:
		fmt.Fprintf(w, "Sorry, only POST methods are supported.")
	}
}

func servicesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		latestServices := getLatestServices()
		latestServicesAsBytes, _ := json.Marshal(latestServices)
		fmt.Fprint(w, string(latestServicesAsBytes))
	default:
		fmt.Fprintf(w, "Sorry, only GET methods are supported.")
	}
}

func serviceLatestStatesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		latestServiceStates := getServiceStatesByName(vars["service_name"])
		formattedList := []serviceStateJson{}

		for i, s := range latestServiceStates {
			var endTime int64
			lookAhead := i + 1

			if len(latestServiceStates) > lookAhead {
				endTimeTimestamp, _ := time.Parse(time.RFC3339, latestServiceStates[lookAhead].Dt)
				endTime = endTimeTimestamp.Unix()
			}

			startTimeTimestamp, _ := time.Parse(time.RFC3339, s.Dt)

			formattedList = append(
				formattedList,
				serviceStateJson{State: s.State, Start_Time: startTimeTimestamp.Unix(), End_Time: endTime})
		}

		formattedListAsBytes, _ := json.Marshal(formattedList)
		fmt.Fprint(w, string(formattedListAsBytes))
	default:
		fmt.Fprintf(w, "Sorry, only GET methods are supported.")
	}
}

func main() {
	var config = Config{
		Host:     getEnv("DB_HOST", "postgres"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASS", "postgres"),
		Database: getEnv("DB_NAME", "postgres"),
	}

	postgres, _ = New(config)
	httpState := NewState()

	r := mux.NewRouter()
	r.HandleFunc("/health", httpState.Health)
	r.HandleFunc("/event", eventHandler)
	r.HandleFunc("/services", servicesHandler)
	r.HandleFunc("/services/{service_name}/latest-states", serviceLatestStatesHandler)
	http.Handle("/", r)

	fmt.Printf("Starting server...\n")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}
