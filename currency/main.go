package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var ADRESS_API = "https://v6.exchangerate-api.com/v6/1d40c79e54dd9bde719c2bb0/latest/USD"
var TIME_DAY = 24 * time.Hour
var DATA_HISTORY map[time.Time]float64

type Data struct {
	Date time.Time
	Val  map[string]float64 `json:"conversion_rates"`
}

func init() {
	DATA_HISTORY = make(map[time.Time]float64)
}

func FoundUSDT() (Data, error) {
	resp, err := http.Get(ADRESS_API)
	if err != nil {
		fmt.Println(err)
		return Data{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return Data{}, err
	}

	answer := &Data{}
	answer.Date = time.Now()
	err = json.Unmarshal(body, answer)
	if err != nil {
		fmt.Println(err)
		return Data{}, err
	}

	DATA_HISTORY[answer.Date] = answer.Val["RUB"]

	return *answer, nil
}

func main() {

	ticker := time.NewTicker(TIME_DAY)

	defer ticker.Stop()

	var rate_usdt float64

	if rate, err := FoundUSDT(); err != nil {
		fmt.Println("Ошибка при получении начального курса RUB:", err)
	} else {
		rate_usdt = rate.Val["RUB"]
	}

	go func() {
		for range ticker.C {
			if rate, err := FoundUSDT(); err != nil {
				fmt.Println("Ошибка при получении начального курса RUB:", err)
			} else {
				rate_usdt = rate.Val["RUB"]
			}
		}
	}()

	http.HandleFunc("/curr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Date: ", time.DateTime)
		fmt.Fprintln(w, "Currency of RUB: ", rate_usdt)
	})

	http.ListenAndServe(":8082", nil)

}
