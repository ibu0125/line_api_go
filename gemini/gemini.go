package gemini

import (
	"context"
	"log"

	"google.golang.org/genai"
)

func AiSystem(incoming_text string)(string,error) {

  	ctx := context.Background()
  	client, err := genai.NewClient(ctx, nil)
  	if err != nil {
      log.Fatal(err)
  	}

	history := []*genai.Content{
    	genai.NewContentFromText(`
			あなたは研究用資料・論文を分析・整理し、WordやPowerPoint向けに分かりやすく文章やスライド内容を作成するAIです。
			以下のルールに従ってください：
			1. 与えられた論文・資料の内容を正確に理解して要約してください。
			2. 重要なポイント、図表、結論を整理して出力してください。
			3. スライドや文書向けに読みやすく構造化してください。
   				- PowerPoint向け: 見出しスライドと箇条書きスライドに分ける
   				- Word向け: セクションごとにタイトルと本文を作成
			4. 文章は専門用語を保持しつつ、必要に応じて簡潔に書き換えてください。
			5. 複数論文を渡された場合は、それぞれ独立して整理してください。
			6. ユーザーから指示があれば、特定のフォーマットに従って出力してください。
			7. 常に一貫性を持たせ、事実誤認のないよう注意してください。
			- ユーザー入力には論文本文や資料を渡すことがあります。
			- ユーザー入力が質問形式の場合は、資料の内容に基づいて回答してください。
			- 出力形式は、後で Word / PowerPoint に変換可能なテキスト形式を優先してください。
		`, "system"),  // ← ここを RoleSystem に変更
	}


  	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, history)
	if err!=nil {
		return "申し訳ございません。うまく読み込めませんでした。　もう一度お願いします",err
	}
	
  	res, err := chat.SendMessage(ctx, genai.Part{Text: incoming_text})
	if err!=nil {
		return "申し訳ございません。うまく読み込めませんでした。　もう一度お願いします",err
	}

  	if len(res.Candidates) > 0 {
	  	ai_reply:=res.Candidates[0].Content.Parts[0].Text
	  	return ai_reply,nil
  	}

	return "申し訳ございません。うまく読み込めませんでした。　もう一度お願いします",err
}