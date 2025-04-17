package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	ADRESS_API   = "https://v6.exchangerate-api.com/v6/1d40c79e54dd9bde719c2bb0/latest/USD"
	TIME_DAY     = 24 * time.Hour
	DATA_HISTORY map[string]float64
	Rate_USD     float64
)

type Currency struct {
	Date time.Time
	Val  map[string]float64 `json:"conversion_rates"`
}

func init() {
	DATA_HISTORY = make(map[string]float64)
}

func FoundUSDT() (Currency, error) {
	resp, err := http.Get(ADRESS_API)
	if err != nil {
		fmt.Println(err)
		return Currency{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return Currency{}, err
	}

	rate := &Currency{}
	rate.Date = time.Now()
	err = json.Unmarshal(body, rate)
	if err != nil {
		fmt.Println(err)
		return Currency{}, err
	}

	return *rate, nil
}

func main() {

	ticker := time.NewTicker(TIME_DAY)

	defer ticker.Stop()

	rate, err := FoundUSDT()

	if err != nil {
		fmt.Println("Ошибка при получении начального курса RUB:", err)
	} else {
		Rate_USD = rate.Val["RUB"]
	}

	go func() {
		for range ticker.C {
			rate, err := FoundUSDT()
			if err != nil {
				fmt.Println("Ошибка при получении начального курса RUB:", err)
			} else {
				Rate_USD = rate.Val["RUB"]
			}
		}
	}()

	http.HandleFunc("/curr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Date: ", time.Now().Format("02.01.2006"))
		fmt.Fprintln(w, "Currency of RUB: ", Rate_USD)
		DATA_HISTORY[rate.Date.Format("02.01.2006")] = rate.Val["RUB"]
	})

	http.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(DATA_HISTORY)
		for key, val := range DATA_HISTORY {
			fmt.Fprintf(w, "Date - %s\nCurrency - %v\n\n", key, val)
		}
	})

	http.ListenAndServe(":8082", nil)

}
