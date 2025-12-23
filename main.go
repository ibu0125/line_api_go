package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

func validateSignature(body []byte, signature string, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}


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


	http.HandleFunc("/callback",func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		signature := r.Header.Get("X-Line-Signature")
		secret := os.Getenv("LINE_CHANNEL_SECRET")

		if !validateSignature(body, signature, secret) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		log.Println("LINE Webhook OK")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

}

