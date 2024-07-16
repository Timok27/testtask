package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

type BankAccount interface {
	Deposit(amount float64) error
	Withdraw(amount float64) error
	GetBalance() float64
}

type Account struct {
	ID      int
	Balance float64
	mutex   sync.Mutex
}

var accounts = make(map[int]*Account)
var idCounter = 1
var mu sync.Mutex

func (a *Account) Deposit(amount float64) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.Balance += amount
	return nil
}

func (a *Account) Withdraw(amount float64) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.Balance < amount {
		return fmt.Errorf("insufficient funds")
	}
	a.Balance -= amount
	return nil
}

func (a *Account) GetBalance() float64 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.Balance
}

func createAccount(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	id := idCounter
	idCounter++
	mu.Unlock()

	account := &Account{ID: id, Balance: 0}
	accounts[id] = account
	log.Printf("Created account with ID: %d at %s", id, time.Now().Format(time.RFC3339))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func deposit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		Amount float64 `json:"amount"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received deposit request: %+v", req)

	account, exists := accounts[id]
	if !exists {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	go func() {
		err := account.Deposit(req.Amount)
		if err != nil {
			log.Printf("Deposit failed for account ID: %d at %s: %v", id, time.Now().Format(time.RFC3339), err)
		} else {
			log.Printf("Deposited %f to account ID: %d at %s", req.Amount, id, time.Now().Format(time.RFC3339))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func withdraw(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		Amount float64 `json:"amount"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received withdraw request: %+v", req)

	account, exists := accounts[id]
	if !exists {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	go func() {
		err := account.Withdraw(req.Amount)
		if err != nil {
			log.Printf("Withdraw failed for account ID: %d at %s: %v", id, time.Now().Format(time.RFC3339), err)
		} else {
			log.Printf("Withdrew %f from account ID: %d at %s", req.Amount, id, time.Now().Format(time.RFC3339))
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func getBalance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	account, exists := accounts[id]
	if !exists {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}

	balance := account.GetBalance()
	log.Printf("Checked balance for account ID: %d at %s: %f", id, time.Now().Format(time.RFC3339), balance)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]float64{"balance": balance})
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/accounts", createAccount).Methods("POST")
	router.HandleFunc("/accounts/{id}/deposit", deposit).Methods("POST")
	router.HandleFunc("/accounts/{id}/withdraw", withdraw).Methods("POST")
	router.HandleFunc("/accounts/{id}/balance", getBalance).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", router))
}
