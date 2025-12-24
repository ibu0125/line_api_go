package main

import (
	"encoding/json"
	"go_project/extraction"
	"go_project/gemini"
	"go_project/supabase"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

var (
	LINE_CHANNEL_SECRET       = os.Getenv("LINE_CHANNEL_SECRET")
	LINE_CHANNEL_ACCESS_TOKEN = os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")

	userMode      = map[string]string{} // chat / generate
	templatePath = map[string]string{} // 保存用（任意）
	templateJSON = map[string]string{} // ★ Word構造JSON
)

func reply(bot *linebot.Client, ev *linebot.Event, text string) {
	_, err := bot.ReplyMessage(
		ev.ReplyToken,
		linebot.NewTextMessage(text),
	).Do()
	if err != nil {
		log.Println("Reply error:", err)
	}
}

func main() {

	bot, err := linebot.New(
		LINE_CHANNEL_SECRET,
		LINE_CHANNEL_ACCESS_TOKEN,
	)
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {

		events, err := bot.ParseRequest(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		for _, ev := range events {

			switch ev.Type {

			// ================= フォロー =================
			case linebot.EventTypeFollow:
				userID := ev.Source.UserID

				exists, err := supabase.IsUser(userID)
				if err != nil {
					reply(bot, ev, "通信エラーが発生しました")
					continue
				}

				if !exists {
					reply(bot, ev, "このAIは購入者限定です。\n認証コードを送信してください")
					continue
				}

				reply(bot, ev, "認証済みです。\n#会話\n#生成\nから選択してください")

			// ================= メッセージ =================
			case linebot.EventTypeMessage:
				userID := ev.Source.UserID

				switch msg := ev.Message.(type) {

				// ---------- テキスト ----------
				case *linebot.TextMessage:
					text := strings.TrimSpace(msg.Text)
					log.Println("TEXT:", userID, text)

					exists, err := supabase.IsUser(userID)
					if err != nil {
						reply(bot, ev, "通信エラーが発生しました")
						continue
					}

					// 未認証
					if !exists {
						ok, err := supabase.UseAuthCode(text)
						if err != nil {
							reply(bot, ev, "通信エラーが発生しました")
							continue
						}

						if ok {
							_ = supabase.AddUser(userID)
							reply(bot, ev, "認証完了しました。\n#会話\n#生成\nを選択してください")
						} else {
							reply(bot, ev, "認証コードが正しくありません")
						}
						continue
					}

					// モード切替
					if text == "#会話" {
						userMode[userID] = "chat"
						reply(bot, ev, "会話モードに切り替えました")
						continue
					}

					if text == "#生成" {
						userMode[userID] = "generate"
						delete(templatePath, userID)
						delete(templateJSON, userID)
						reply(bot, ev, "生成モードです。\nWordテンプレート（.docx）を送信してください")
						continue
					}

					mode := userMode[userID]
					if mode == "" {
						reply(bot, ev, "#会話 または #生成 を選択してください")
						continue
					}

					// ---------- 会話モード ----------
					if mode == "chat" {
						out, err := gemini.ChatAiSystem(text)
						if err != nil {
							reply(bot, ev, "AI応答に失敗しました")
							continue
						}
						reply(bot, ev, out)
						continue
					}

					// ---------- 生成モード ----------
					if mode == "generate" {

						if templateJSON[userID] == "" {
							reply(bot, ev, "先に Wordテンプレート（.docx）を送信してください")
							continue
						}

						out, err := gemini.GenerateAiSystem(
							templateJSON[userID],
							text,
						)
						if err != nil {
							log.Println(err)
							reply(bot, ev, "生成に失敗しました")
							continue
						}

						reply(bot, ev, out)
						continue
					}

				// ---------- Wordファイル ----------
				case *linebot.FileMessage:
					log.Println("FILE:", msg.FileName, userID)

					if userMode[userID] != "generate" {
						reply(bot, ev, "ファイル送信は生成モードで行ってください")
						continue
					}

					if !strings.HasSuffix(strings.ToLower(msg.FileName), ".docx") {
						reply(bot, ev, "対応しているのは Word（.docx）のみです")
						continue
					}

					content, err := bot.GetMessageContent(msg.ID).Do()
					if err != nil {
						reply(bot, ev, "ファイル取得に失敗しました")
						continue
					}
					defer content.Content.Close()

					path := "/tmp/" + userID + "_template.docx"
					f, err := os.Create(path)
					if err != nil {
						reply(bot, ev, "ファイル保存に失敗しました")
						continue
					}
					defer f.Close()

					if _, err := io.Copy(f, content.Content); err != nil {
						reply(bot, ev, "ファイル書き込みに失敗しました")
						continue
					}

					// ★ Word構造抽出
					docStruct, err :=extraction.ExtractWordStructure(path)
					if err != nil {
						reply(bot, ev, "Word構造の解析に失敗しました")
						continue
					}

					jsonBytes, _ := json.MarshalIndent(docStruct, "", "  ")
					templateJSON[userID] = string(jsonBytes)
					templatePath[userID] = path

					reply(bot, ev,
						"✅ Wordテンプレートを解析しました\n"+
							"次に【研究内容】を送信してください",
					)
				}
			}
		}
	})

	log.Println("Listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
