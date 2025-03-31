package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Data struct {
	Curr string             `json:"date"`
	Val  map[string]float64 `json:"rub"`
}

func FoundUSDT() float64 {
	resp, err := http.Get("https://latest.currency-api.pages.dev/v1/currencies/rub.json")
	if err != nil {
		fmt.Println(err)
		return 0
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	answer := &Data{}
	err = json.Unmarshal(body, answer)
	if err != nil {
		fmt.Println(err)
		return 0
	}

	return answer.Val["usdt"]
}

func main() {

	ticker := time.NewTicker(24 * time.Hour)

	rate_usdt := FoundUSDT()

	select {
	case <-ticker.C:
		rate_usdt = FoundUSDT()
	}

	http.HandleFunc("/curr", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Currency of USTD: ", rate_usdt)
	})

	http.ListenAndServe(":8082", nil)

}
