package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"
)



func validateSignature(body []byte, signature string, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}


func main() {

	LINE_CHANNEL_SECRET:=os.Getenv("LINE_CHANNEL_SECRET")
	LINE_CHANNEL_ACCESS_TOKEN:=os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	bot,err:=linebot.New(LINE_CHANNEL_SECRET,LINE_CHANNEL_ACCESS_TOKEN)
	if err !=nil{
		fmt.Printf("",err)
	}


	port := os.Getenv("PORT")
	if port=="" {
		port="8080"
	}


	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/callback",func(w http.ResponseWriter, r *http.Request) {
        events, err := bot.ParseRequest(r)
        if err != nil {
            if err == linebot.ErrInvalidSignature {
                w.WriteHeader(http.StatusBadRequest)
            } else {
                w.WriteHeader(http.StatusInternalServerError)
            }
            log.Printf("ParseRequest error: %v", err)
            return
        }

		for _,ev:=range events{
			switch ev.Type{
				case linebot.EventTypeFollow:
					userId:=ev.Source.UserID

					if _,err:=bot.ReplyMessage(ev.ReplyToken,linebot.NewTextMessage("よろしく")).Do();err!=nil{
						log.Printf("ReplyMessage error: %v", err)
					}
			}


		}


	})

}

