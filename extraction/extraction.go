package extraction

import (
	"log"
	"os"
	"strings"

	"github.com/unidoc/unioffice/common/license"
	"github.com/unidoc/unioffice/document"
	"github.com/unidoc/unioffice/measurement"
	"github.com/unidoc/unioffice/schema/soo/wml"
)

/* =======================
   構造体定義
======================= */

type DocTemplate struct {
	Type     string    `json:"type"`
	Sections []Section `json:"sections"`
}

type Section struct {
	Title *Block  `json:"title,omitempty"`
	Body  []Block `json:"body"`
}

type Block struct {
	Kind   string  `json:"kind"` 
	Style  string  `json:"style,omitempty"`
	Indent int     `json:"indent,omitempty"`
	Runs   []Run   `json:"runs,omitempty"`
	Items  [][]Run `json:"items,omitempty"`
}

type Run struct {
	Text     string `json:"text"`
	Bold     bool   `json:"bold,omitempty"`
	Italic   bool   `json:"italic,omitempty"`
	FontSize int    `json:"fontSize,omitempty"`
}



func init() {
    key := os.Getenv("UNICLOUD_API_KEY")
    err := license.SetMeteredKey(key) 
    if err != nil {
        log.Fatalf("UniOffice ライセンス設定失敗: %v", err)
    }
}

/* =======================
   Word → JSON 抽出
======================= */

func paragraphText(p document.Paragraph) string {
	var sb strings.Builder
	for _, r := range p.Runs() {
		sb.WriteString(r.Text())
	}
	return sb.String()
}

func getFontSize(r document.Run) int {
	props := r.Properties()
	if props.X() == nil || props.X().Sz == nil {
		return 0
	}
	sz := props.X().Sz.ValAttr
	if sz.ST_UnsignedDecimalNumber != nil {
		return int(*sz.ST_UnsignedDecimalNumber / 2) // half-point → pt
	}
	return 0
}

func ExtractWordStructure(path string) (*DocTemplate, error) {
	doc, err := document.Open(path)
	if err != nil {
		return nil, err
	}

	result := &DocTemplate{Type: "word"}
	var current *Section
	var currentList *Block
	var currentListID int64 = -1

	for _, p := range doc.Paragraphs() {
		text := paragraphText(p)

		// 空行
		if strings.TrimSpace(text) == "" {
			currentList = nil
			currentListID = -1
			if current != nil {
				current.Body = append(current.Body, Block{Kind: "blank_line"})
			}
			continue
		}

		// 見出し
		if strings.HasPrefix(p.Style(), "Heading") {
			currentList = nil
			currentListID = -1
			sec := Section{Title: extractParagraphBlock(p)}
			result.Sections = append(result.Sections, sec)
			current = &result.Sections[len(result.Sections)-1]
			continue
		}

		if current == nil {
			result.Sections = append(result.Sections, Section{})
			current = &result.Sections[len(result.Sections)-1]
		}

		// 箇条書き
		pp := p.Properties().X()
		if pp != nil && pp.NumPr != nil {
			numID := int64(0)
			level := 0
			if pp.NumPr.NumId != nil {
				numID = int64(pp.NumPr.NumId.ValAttr)
			}
			if pp.NumPr.Ilvl != nil {
				level = int(pp.NumPr.Ilvl.ValAttr)
			}

			if currentList == nil || currentListID != numID {
				list := Block{Kind: "list", Indent: level}
				current.Body = append(current.Body, list)
				currentList = &current.Body[len(current.Body)-1]
				currentListID = numID
			}

			item := []Run{}
			for _, r := range p.Runs() {
				item = append(item, Run{
					Text:     r.Text(),
					Bold:     r.Properties().IsBold(),
					Italic:  r.Properties().IsItalic(),
					FontSize: getFontSize(r),
				})
			}
			currentList.Items = append(currentList.Items, item)
			continue
		}

		// 通常段落
		currentList = nil
		currentListID = -1
		current.Body = append(current.Body, *extractParagraphBlock(p))
	}

	return result, nil
}

func extractParagraphBlock(p document.Paragraph) *Block {
	block := &Block{Kind: "paragraph", Style: p.Style()}
	for _, r := range p.Runs() {
		block.Runs = append(block.Runs, Run{
			Text:     r.Text(),
			Bold:     r.Properties().IsBold(),
			Italic:  r.Properties().IsItalic(),
			FontSize: getFontSize(r),
		})
	}
	return block
}

/* =======================
   JSON → Word 再構築
======================= */

func ApplyJSONToWordStruct(template *DocTemplate, outputPath string) error {
	doc := document.New()

	// 箇条書き用 NumberingDefinition
	numDef := createBulletNumbering(doc)

	for _, sec := range template.Sections {
		if sec.Title != nil {
			p := doc.AddParagraph()
			p.SetStyle(sec.Title.Style)
			applyRuns(p, sec.Title.Runs)
		}

		for _, b := range sec.Body {
			switch b.Kind {
			case "blank_line":
				doc.AddParagraph()

			case "paragraph":
				p := doc.AddParagraph()
				p.SetStyle(b.Style)
				applyRuns(p, b.Runs)

			case "list":
				for _, item := range b.Items {
					p := doc.AddParagraph()
					p.SetNumberingDefinition(numDef)
					p.SetNumberingLevel(0)
					applyRuns(p, item)
				}
			}
		}
	}

	return doc.SaveToFile(outputPath)
}

func applyRuns(p document.Paragraph, runs []Run) {
	for _, r := range runs {
		run := p.AddRun()
		run.AddText(r.Text)
		run.Properties().SetBold(r.Bold)
		run.Properties().SetItalic(r.Italic)
		if r.FontSize > 0 {
			run.Properties().SetSize(measurement.Distance(r.FontSize * 2)) // pt → half-point
		}
	}
}

/* =======================
   箇条書き定義
======================= */

func createBulletNumbering(doc *document.Document) document.NumberingDefinition {
	numbering := doc.Numbering
	numDef := numbering.AddDefinition() // struct が返る（ポインタ不要）

	lvl := numDef.AddLevel()
	lvl.SetFormat(wml.ST_NumberFormatBullet)
	lvl.SetText("•")
	lvl.Properties().SetLeftIndent(measurement.Distance(720))
	lvl.Properties().SetHangingIndent(measurement.Distance(360))

	return numDef // ポインタではなく struct のまま返す
}

