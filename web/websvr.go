package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"

	"regexp"

	"html/template"
	"net/http"

	"time"

	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

const (
	port = ":1664"

	templateDir  = "tmpl/"
	templateView = templateDir + "view.html"

	apiKeyFile = "apikey.txt"

	dbFile = "./data.db"

	paramApiKey    = "apiKey"
	paramTimestamp = "d"
	paramTemp      = "t"
	paramHumidity  = "h"
	paramEvent     = "e"
)

var (
	events map[string]string = map[string]string{
		"DO": "",
		"DC": "",
	}
	templ  = template.Must(template.ParseFiles(templateView))
	apiKey string
	db     *sql.DB
)

type structEvent struct {
	Timestamp time.Time
	Event     string
}
type structMeasurement struct {
	Timestamp             time.Time
	Temperature, Humidity float64
}

type templateData struct {
	TemperatureValues [][]string
	HumidityValues    [][]string
	Events            [][]string
}

func loadApiKey() {
	body, err := ioutil.ReadFile(apiKeyFile)
	if err != nil {
		panic(err)
	}

	if match, _ := regexp.Match("^[0-9a-f]{40}$", body); !match {
		panic("Content of file [" + apiKeyFile + "] is not a valid SHA-1 hash.")
	}

	apiKey = string(body)
}

func openDb() *sql.DB {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatal(err)
	}

	sqlCreate := `
		CREATE TABLE IF NOT EXISTS measurements (
			tstamp timestamp not null primary key, 
			temperature float, 
			humidity float);
		CREATE TABLE IF NOT EXISTS events (
			tstamp timestamp not null primary key, 
			type text);
		`
	_, err = db.Exec(sqlCreate)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlCreate)
		db.Close()
		panic("Failed creating tables.")
	}

	return db
}

func main() {
	loadApiKey()
	db = openDb()
	defer db.Close()

	http.Handle("/", http.HandlerFunc(View))
	http.Handle("/add", http.HandlerFunc(Add))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func View(w http.ResponseWriter, req *http.Request) {
	log.Println("Call to View")

	rows, err := db.Query(`SELECT "m", strftime('%s', tstamp)*1000 ts, temperature, humidity 
				FROM measurements 
			UNION
				SELECT "e", strftime('%s', tstamp)*1000 ts, type, ""
				FROM events 
			ORDER BY ts ASC`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	temperatureValues := [][]string{}
	humidityValues := [][]string{}
	events := [][]string{}
	do := ""
	dc := ""

	for rows.Next() {
		var table, tstamp, data1, data2 string
		rows.Scan(&table, &tstamp, &data1, &data2)

		switch table {
		case "m":
			temperatureValues = append(temperatureValues, []string{tstamp, data1})
			humidityValues = append(humidityValues, []string{tstamp, data2})
		case "e":
			switch data1 {
			case "DO":
				if len(do) > 0 {
					log.Printf("WARN: will ignore encountered event DO at %s already set at %s\n", tstamp, do)
				} else {
					do = tstamp
				}
			case "DC":
				if len(do) == 0 {
					log.Printf("WARN: will ignore encountered event DC at %s although DO is not set\n", tstamp)
				} else {
					dc = tstamp
				}
			}

			if len(do) > 0 && len(dc) > 0 {
				events = append(events, []string{do, dc})
				do = ""
				dc = ""
			}
		}

	}

	templ.Execute(w, templateData{temperatureValues, humidityValues, events})
}

func Add(w http.ResponseWriter, req *http.Request) {
	log.Printf("Call to Add with query string: %s\n", req.URL.RawQuery)

	err := req.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	log.Println(req.Form)

	if key, ok := req.Form[paramApiKey]; !ok || key[0] != apiKey {
		http.Error(w, "Api Key is not valid.", http.StatusForbidden)
		return
	}

	measurementsToAdd := []structMeasurement{}
	eventsToAdd := []structEvent{}
	for i, ts := range req.Form[paramTimestamp] {
		tsInt, err := strconv.ParseInt(ts, 0, 64)
		if err != nil {
			log.Print("Error parsing timestamp: ")
			log.Println(err)
			continue
		}
		time := time.Unix(tsInt, 0)

		var t float64 = 0
		if len(req.Form[paramTemp]) > i && req.Form[paramTemp][i] != "" {
			t, err = strconv.ParseFloat(req.Form[paramTemp][i], 64)
			if err != nil {
				log.Printf("Error parsing temperature '%s' for timestamp %s\n\t", req.Form[paramTemp][i], ts)
				log.Println(err)
			}
		}

		var h float64 = 0
		if len(req.Form[paramHumidity]) > i && req.Form[paramHumidity][i] != "" {
			h, err = strconv.ParseFloat(req.Form[paramHumidity][i], 64)
			if err != nil {
				log.Printf("Error parsing humidity '%s' for timestamp %s\n\t", req.Form[paramHumidity][i], ts)
				log.Println(err)
			}
		}

		if t > 0 || h > 0 {
			measurementsToAdd = append(measurementsToAdd, structMeasurement{time, t, h})
		}

		if len(req.Form[paramEvent]) > i && req.Form[paramEvent][i] != "" {
			e := req.Form[paramEvent][i]
			if _, ok := events[e]; !ok {
				log.Printf("Unknown event '%s' for timestamp %s\n", e, ts)
			} else {
				eventsToAdd = append(eventsToAdd, structEvent{time, e})
			}
		}
	}

	mDone, eDone := addDataToDb(measurementsToAdd, eventsToAdd)

	fmt.Fprintf(w, "Added %d measurements and %d events.", mDone, eDone)
}

func addDataToDb(measurements []structMeasurement, events []structEvent) (mDone, eDone int) {
	if measurements == nil || events == nil {
		panic("Parameters slices cannot be nil.")
	}

	if len(measurements)+len(events) == 0 {
		log.Println("No data to save.")
	}

	log.Printf("%d measurements and %d events to add\n", len(measurements), len(events))

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	if len(measurements) > 0 {
		stmt, err := tx.Prepare("INSERT INTO measurements(tstamp, temperature, humidity) VALUES(?, ?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		for _, m := range measurements {
			_, err = stmt.Exec(m.Timestamp, m.Temperature, m.Humidity)
			if err != nil {
				log.Println(err)
			} else {
				mDone++
			}
		}
	}

	if len(events) > 0 {
		stmt, err := tx.Prepare("INSERT INTO events(tstamp, type) VALUES(?, ?)")
		if err != nil {
			log.Fatal(err)
		}
		defer stmt.Close()

		for _, e := range events {
			_, err := stmt.Exec(e.Timestamp, e.Event)
			if err != nil {
				log.Println(err)
			} else {
				eDone++
			}
		}
	}

	tx.Commit()

	return mDone, eDone
}
