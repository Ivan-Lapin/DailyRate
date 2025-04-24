package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/Ivan-Lapin/DailyRate/proto/currency/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ADRESS_API   = "https://v6.exchangerate-api.com/v6/1d40c79e54dd9bde719c2bb0/latest/USD"
	TIME_DAY     = 24 * time.Hour
	DATA_HISTORY map[string]float64
	Rate_USD     float64
)

type server struct {
	pb.UnimplementedCurrencyServiceServer
}

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

func (s *server) GetCurrentRate(ctx context.Context, req *pb.GetCurrentRateRequest) (*pb.GetCurrentRateResponse, error) {
	rate, err := FoundUSDT()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch rate: %v", err)
	}

	Rate_USD = rate.Val["RUB"]
	DATA_HISTORY[rate.Date.Format("02.01.2006")] = rate.Val["RUB"]

	return &pb.GetCurrentRateResponse{
		Date: time.Now().Format("02.01.2006"),
		Rate: Rate_USD,
	}, nil

}

func (s *server) GetHistoryRate(ctx context.Context, req *pb.GetHistoryRateRequest) (*pb.GetHistoryRateResponse, error) {
	return &pb.GetHistoryRateResponse{
		History: DATA_HISTORY,
	}, nil
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

	lis, err := net.Listen("tcp", "localhost:8082")
	if err != nil {
		fmt.Printf("failed to listen: %v\n", err)
		return
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCurrencyServiceServer(grpcServer, &server{})
	fmt.Println("gRPC server is running on :8082")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("failed to serve: %v\n", err)
	}

}
