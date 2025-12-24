package main

import (
	"go_project/gemini"
	"go_project/supabase"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"
)

var(
    LINE_CHANNEL_SECRET = os.Getenv("LINE_CHANNEL_SECRET")
    LINE_CHANNEL_ACCESS_TOKEN = os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
)

func reply(bot *linebot.Client, ev *linebot.Event, replyText string) {
	_, err := bot.ReplyMessage(
		ev.ReplyToken,
		linebot.NewTextMessage(replyText),
	).Do()

	if err != nil {
		log.Printf("ReplyMessage error: %v", err)
	}
}


func main() {
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
					reply(bot,ev,"通信エラーが発生しました。しばらくしてからもう一度お試しください。")
                    continue
                }

                if !exists {
					reply(bot,ev,"このAIは購入者限定です。認証コードを送信してください")
                    continue
                }else{
					profile, err := bot.GetProfile(userId).Do()
                	if err != nil || profile == nil {
                    	log.Printf("GetProfile error: %v", err)
						reply(bot,ev,"フォローありがとうございます！よろしくお願いいたします。")
               	 	}

                	displayName := profile.DisplayName
					reply(bot,ev,"認証完了しました。"+displayName+"様、よろしくお願いいたします")

					continue
				}



			case linebot.EventTypeMessage:
				userId:=ev.Source.UserID
				switch message:=ev.Message.(type){
				case *linebot.TextMessage:
					incoming_text:=message.Text
					log.Println("受信したテキスト：",incoming_text,userId)

					exists,err:=supabase.IsUser(userId)

					if err != nil {
                    	log.Printf("IsUser error: %v", err)
						reply(bot,ev,"通信エラーが発生しました。しばらくしてからもう一度お試しください。")
                    	continue
                	}

					log.Println("user:",exists)
					if !exists {
						exists_code,err:=supabase.UseAuthCode(incoming_text)
						if err != nil {
                    		log.Printf("IsUser error: %v", err)
							reply(bot,ev,"通信エラーが発生しました。しばらくしてからもう一度お試しください。")
                			continue
            			}

						log.Println("code:",exists_code)

						if exists_code {
							profile, err := bot.GetProfile(userId).Do()
                			if err != nil || profile == nil {
                    			log.Printf("GetProfile error: %v", err)
								reply(bot,ev,"フォローありがとうございます！よろしくお願いいたします。")
               	 			}

                			displayName := profile.DisplayName
							reply(bot,ev,"認証完了しました。"+displayName+"様、よろしくお願いいたします")
							continue
						}else{
							reply(bot,ev,"このAIは購入者限定です。認証コードを送信してください")
                    		continue
						}
					}else{

						//文章をAIに渡す
						reply_text,err:=gemini.AiSystem(incoming_text)
						if err!=nil {
							reply(bot,ev,reply_text)
							continue
						}
						reply(bot,ev,reply_text)
						continue
					}
				}
            }
        }
    })
    log.Printf("Listening on :%s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
