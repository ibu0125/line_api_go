package gemini

import (
	"context"
	"encoding/json"
	"go_project/extraction"
	"log"

	"google.golang.org/genai"
)


func ChatAiSystem(incomingText string) (string, error) {
	ctx := context.Background()

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 🔹 system 相当の指示は「最初の user メッセージ」として入れる
	history := []*genai.Content{
		genai.NewContentFromText(
			"あなたはユーザーの要望に応える会話aiです。普通の会話だけでなく、調べ物や計算なども行ってください。"+
			"また何かを評価するときは厳しく、それ以外は肯定しつつ応対してください。"+
			"名前は2次元AIメイドさやかちゃんです。",
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
    client, err := genai.NewClient(ctx, nil)
    if err != nil {
        log.Fatal(err)
		log.Println(err)
    }

   systemPrompt := "あなたは「文書構造再現AI」です。\n" +
"以下に渡される JSON は、Word 文書から抽出した構造テンプレートです。\n" +
"この JSON は段落・見出し・箇条書き・空行・装飾を厳密に表しています。\n" +
"以下書き方ルール\n\n" +
"【原稿の書き方】\n" +
"2.原稿用紙のサイズ\n" +
"A4版の白紙に,上 15mm,下 20mm,左右 20mmの余白をとり,横書きで53文字×60行で設定する.\n" +
"一人あたりA4で 2 ページに納める.\n" +
"3.表題など\n" +
"・和文表題は12ポイントの文字を使用し,中央揃えとし,ゴシックフォントを使用する.\n" +
"・英文表題は 9 ポイントの文字を使用し,和文表題の下に中央揃えとする.\n" +
"・学生番号,氏名,所属研究室は 1 行に書き,12ポイントの文字(明朝体)を使用し,中央揃えとする.\n" +
"・1 ページ目左上に,平成 21 年度卒業研究発表会(日本大学工学部情報工学科),その下に記号#1-#2 を明朝体 9 ポイントで記述する.\n" +
"ただし,#1 は会場番号で#2 は会場での発表番号である.2 ページ目右上に平成 21 年度卒業研究発表会:記号#1-#2 を明朝体 9 ポイントで記述する.\n" +
"4.本文\n" +
"2 段組とし,1 行の文字数は 25 文字で,明朝体 9 ポイントを使用する.2 ページ目の先頭行は右段・左段ともに 3 行目から書く.\n" +
"5.各節の表題\n" +
"表題の前後に 0.5 行の間隔をあける(表題を選択し,ワードメニューの書式→段落→インデントの行間隔で間隔の段落前・段落後を 0.5 行とする).\n" +
"ゴシックフォントを使用し,書き始めは 1 文字分空白を入れる.\n" +
"6.図表・写真・表\n" +
"図,写真番号は図 1,図 2,・・・,表番号は表 1,表 2,・・・のように記載する.\n" +
"図・写真のタイトルは,図の下側に,表のタイトルは表の上側にゴシックフォントを使用し,中央揃えとする.\n" +
"7.式\n" +
"式番号は,(1),(2),・・・・のように記載し,右揃えとする.式と文章の間に空白を入れる.\n" +
"本文中では,「式(1)は・・・を表す」のように記述する.\n" +
"8.参考文献\n" +
"本文中の引用箇所には,文章右肩に(上付き添え字で)小括弧[ ]を付した番号を記入し,同じ番号で要旨末尾に文献内容(著者名,題目,出展名,ページ,発行年月日)を記載する.\n\n" +
"【最重要ルール】\n" +
"- JSON の構造は一切変更してはいけません\n" +
"- フィールドの追加・削除・順序変更は禁止\n" +
"- kind / indent / style / bold / italic / fontSize は変更禁止\n" +
"- 改行・空行・箇条書きレベルは必ず維持してください\n" +
"- 出力は JSON のみ\n" +
"- Markdown や自然文は禁止\n" +
"- runs 配列の要素数は必ず元と同じにしてください\n" +
"- runs 配列を空にしてはいけません\n\n" +
"【書き換え許可】\n" +
"- runs[].text の文字列のみ変更可能\n\n" +
"以下のメッセージに構造テンプレートJSONと研究内容が同時に渡されます。\n" +
"研究内容を用いて、runs[].text の内容のみを書き換えてください。"


    chat, err := client.Chats.Create(ctx, "gemini-2.5-flash", nil, []*genai.Content{
        genai.NewContentFromText(systemPrompt, "user"),
    })
    if err != nil {
		println(err)
        return "初期化失敗", err
    }

   userPrompt := "【構造テンプレートJSON】\n" + templateJSON + "\n【新しい研究内容】\n" + researchText


    res, err := chat.SendMessage(ctx, genai.Part{Text: userPrompt})
    if err != nil {
		println(err)
        return "生成失敗", err
    }

    if len(res.Candidates) == 0 || len(res.Candidates[0].Content.Parts) == 0 {
		log.Println(err)
        return "応答なし", nil
    }

    aiJSON := res.Candidates[0].Content.Parts[0].Text

    var newTemplate extraction.DocTemplate
    err = json.Unmarshal([]byte(aiJSON), &newTemplate)
    if err != nil {
		println(err)
        return "JSONパース失敗", err
    }

    outputPath := "output.docx"
    err = extraction.ApplyJSONToWordStruct(&newTemplate, outputPath)
    if err != nil {
		println(err)
        return "Word書き出し失敗", err
    }

    return outputPath, nil
}
