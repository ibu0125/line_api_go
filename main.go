package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port=="" {
		port="8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})


	http.HandleFunc("/hello",func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		res:=map[string]string{
			"Message":"Hello API",
			"Status":http.StatusText(http.StatusOK),
		}

		w.Header().Set("Content-Type","application/json")
		err:=json.NewEncoder(w).Encode(res)
		if err != nil {
			log.Printf("Json Err: %v",err)
		}
	})

	log.Fatal(http.ListenAndServe(":"+port,nil))
}