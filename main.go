package main

import (
	"go_project/supabase"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"
)

func main() {
    LINE_CHANNEL_SECRET := os.Getenv("LINE_CHANNEL_SECRET")
    LINE_CHANNEL_ACCESS_TOKEN := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")

    bot, err := linebot.New(LINE_CHANNEL_SECRET, LINE_CHANNEL_ACCESS_TOKEN)
    if err != nil {
        log.Fatalf("linebot.New error: %v", err)
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
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

        for _, ev := range events {
            switch ev.Type {
            case linebot.EventTypeFollow:
                userId := ev.Source.UserID

                exists, err := supabase.IsUser(userId)
                if err != nil {
                    log.Printf("IsUser error: %v", err)
                    if _, err := bot.ReplyMessage(
                        ev.ReplyToken,
                        linebot.NewTextMessage("通信エラーが発生しました。しばらくしてからもう一度お試しください。"),
                    ).Do(); err != nil {
                        log.Printf("ReplyMessage error: %v", err)
                    }
                    continue
                }

                if !exists {
                    if _, err := bot.ReplyMessage(
                        ev.ReplyToken,
                        linebot.NewTextMessage("このAIは購入者限定です。認証コードを送信してください"),
                    ).Do(); err != nil {
                        log.Printf("ReplyMessage error: %v", err)
                    }
                    continue
                }

                profile, err := bot.GetProfile(userId).Do()
                if err != nil || profile == nil {
                    log.Printf("GetProfile error: %v", err)
                    if _, err := bot.ReplyMessage(
                        ev.ReplyToken,
                        linebot.NewTextMessage("フォローありがとうございます！よろしくお願いいたします。"),
                    ).Do(); err != nil {
                        log.Printf("ReplyMessage error: %v", err)
                    }
                    continue
                }

                displayName := profile.DisplayName
                if _, err := bot.ReplyMessage(
                    ev.ReplyToken,
                    linebot.NewTextMessage(displayName+"様、よろしくお願いいたします"),
                ).Do(); err != nil {
                    log.Printf("ReplyMessage error: %v", err)
                }
            }
        }
    })

    log.Printf("Listening on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
