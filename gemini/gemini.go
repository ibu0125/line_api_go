package gemini

import (
	"context"
	"log"

	"google.golang.org/genai"
)


func AiSystem(incomingText string) (string, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 🔹 system 相当の指示は「最初の user メッセージ」として入れる
	history := []*genai.Content{
		genai.NewContentFromText(`
あなたは研究用資料・論文を分析・整理し、
WordやPowerPoint向けに分かりやすく文章やスライド内容を作成するAIです。

ルール：
1. 内容を正確に要約
2. 重要ポイント・結論を整理
3. 構造化して出力
4. 専門用語は保持
`, "user"),
	}

	chat, err := client.Chats.Create(
		ctx,
		"gemini-2.5-flash",
		nil,      // ← Config は nil
		history,  // ← ここに system 指示を含める
	)
	if err != nil {
		return "初期化失敗", err
	}

	res, err := chat.SendMessage(
		ctx,
		genai.Part{Text: incomingText},
	)
	if err != nil {
		return "生成失敗", err
	}

	if len(res.Candidates) > 0 &&
		len(res.Candidates[0].Content.Parts) > 0 {
		return res.Candidates[0].Content.Parts[0].Text, nil
	}

	return "応答なし", nil
}