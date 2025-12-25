package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go_project/extraction"
	"log"
	"os"
	"regexp"
	"strings"

	"google.golang.org/genai"
)

var apiKey=os.Getenv("GEMINI_API_KEY")

func cleanJSONFromText(s string) (string, error) {
    // よくあるパターン：```json ... ``` を取り除く
    s = strings.TrimSpace(s)
    if strings.HasPrefix(s, "```") {
        // 最初のフェンスを除去
        // 例: ```json\n{...}\n```
        s = strings.TrimPrefix(s, "```json")
        s = strings.TrimPrefix(s, "```JSON")
        s = strings.TrimPrefix(s, "```")
        // 終端フェンス除去
        if idx := strings.LastIndex(s, "```"); idx >= 0 {
            s = s[:idx]
        }
        s = strings.TrimSpace(s)
    }

    // 先頭・末尾のバッククォートや不要文字を削除
    s = strings.Trim(s, "` \t\r\n")

    // 正規表現で最初の JSON オブジェクト/配列を抽出
    re := regexp.MustCompile(`(?s)(\{.*\}|\[.*\])`)
    m := re.FindString(s)
    if m == "" {
        return "", errors.New("JSON本体が見つかりません（出力に説明文が混在）")
    }
    return m, nil
}


func ChatAiSystem(incomingText string) (string, error) {
	ctx := context.Background()

	
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  apiKey,
        Backend: genai.BackendGeminiAPI,
    })

	if err != nil {
		log.Fatal(err)
	}

	// 🔹 system 相当の指示は「最初の user メッセージ」として入れる
	history := []*genai.Content{
		genai.NewContentFromText(
			"あなたはユーザーの要望に応える会話AIです。普通の会話だけでなく、調べ物や計算も行ってください。名前は2次元AIメイドさやかちゃんです。",
		"user"),
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



func GenerateAiSystem(templateJSON string, researchText string) (string, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
        APIKey:  apiKey,
        Backend: genai.BackendGeminiAPI,
    })
	
	if err != nil {
		return "", fmt.Errorf("Gemini初期化失敗: %v", err)
	}

	// prompt.txtを読み込む
	systemPromptBytes, err := os.ReadFile("prompt.txt")
	if err != nil {
		return "", fmt.Errorf("prompt.txt読み込み失敗: %v", err)
	}
	systemPrompt := string(systemPromptBytes)

	chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, []*genai.Content{
		genai.NewContentFromText(systemPrompt, "user"),
	})
	if err != nil {
		return "初期化失敗", err
	}

	userPrompt := "【構造テンプレートJSON】\n" + templateJSON + "\n【新しい研究内容】\n" + researchText

	res, err := chat.SendMessage(ctx, genai.Part{Text: userPrompt})
	if err != nil {
    	return "生成失敗", err
	}

	// Candidates[0] のテキストをクリーンに
	aiRaw := res.Candidates[0].Content.Parts[0].Text
	aiJSON, err := cleanJSONFromText(aiRaw)
	if err != nil {
    	log.Printf("AI生出力: %q", aiRaw)
    	return "", fmt.Errorf("JSON抽出失敗: %w", err)
	}

	var newTemplate extraction.DocTemplate
	if err := json.Unmarshal([]byte(aiJSON), &newTemplate); err != nil {
    	return "JSONパース失敗", err
	}

	outputPath := os.TempDir() + "/output.docx"
	if err := extraction.ApplyJSONToWordStruct(&newTemplate, outputPath); err != nil {
    	return "Word書き出し失敗", err
	}

	return outputPath, nil

}


func ChatWithHistory(
	history []map[string]string,
	userText string,
) (string, error) {

	ctx := context.Background()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", err
	}

	contents := []*genai.Content{
		genai.NewContentFromText(
			"あなたは会話を記憶するAIメイド『さやかちゃん』です。",
			genai.RoleUser,
		),
	}

	for _, m := range history {

		var role genai.Role
		switch m["role"] {
		case "user":
			role = genai.RoleUser
		case "assistant":
			role = genai.RoleModel
		default:
			continue
		}

		contents = append(contents,
			genai.NewContentFromText(
				m["content"],
				role,
			),
		)
	}

	chat, err := client.Chats.Create(
		ctx,
		"gemini-2.5-flash",
		nil,
		contents,
	)
	if err != nil {
		return "", err
	}

	res, err := chat.SendMessage(
		ctx,
		genai.Part{Text: userText},
	)
	if err != nil {
		return "", err
	}

	return res.Candidates[0].Content.Parts[0].Text, nil
}

