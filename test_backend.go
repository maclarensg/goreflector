//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		response := map[string]interface{}{
			"message":   "Hello from test backend",
			"path":      r.URL.Path,
			"method":    r.Method,
			"headers":   r.Header,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend", "test-backend")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("slow response"))
	})

	http.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("intentional error"))
	})

	fmt.Println("Test backend starting on :9999")
	log.Fatal(http.ListenAndServe(":9999", nil))
}
