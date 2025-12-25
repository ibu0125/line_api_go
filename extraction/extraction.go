package extraction

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/unidoc/unioffice/common"
	"github.com/unidoc/unioffice/common/license"
	"github.com/unidoc/unioffice/document"
	"github.com/unidoc/unioffice/measurement"
	"github.com/unidoc/unioffice/schema/soo/wml"
)

/* =======================
   構造体
======================= */

type DocTemplate struct {
    Type         string       `json:"type"`
    Sections     []Section    `json:"sections"`
    PageSettings *wml.CT_SectPr  `json:"-"` // 用紙サイズ・余白情報を保持
}
type Section struct {
    Title *Block  `json:"title,omitempty"`
    Body  []Block `json:"body"`
}
type Block struct {
    Kind   string      `json:"kind"` // paragraph, list, table, image, blank_line
    Style  string      `json:"style,omitempty"`
    Indent int         `json:"indent,omitempty"`
    Runs   []Run       `json:"runs,omitempty"`
    Items  [][]Run     `json:"items,omitempty"`
    Rows   [][]Block   `json:"rows,omitempty"`
    Image  *ImageBlock `json:"image,omitempty"`
    Column int         `json:"column,omitempty"`
}
type Run struct {
    Text      string `json:"text"`
    Bold      bool   `json:"bold,omitempty"`
    Italic    bool   `json:"italic,omitempty"`
    FontSize  int    `json:"fontSize,omitempty"`
    Hyperlink string `json:"hyperlink,omitempty"`
}
type ImageBlock struct {
    Name string `json:"name"`
    Data []byte `json:"data"`
}

/* =======================
   ライセンス
======================= */

func init() {
    key := os.Getenv("UNICLOUD_API_KEY")
    if err := license.SetMeteredKey(key); err != nil {
        log.Fatalf("UniOffice ライセンス設定失敗: %v", err)
    }
}


/* =======================
   ページ設定抽出
======================= */
func extractPageSettings(doc *document.Document) *wml.CT_SectPr {
    sectPr := doc.X().Body.SectPr
    if sectPr == nil {
        return nil
    }
    // コピーして返す
    copy := *sectPr
    return &copy
}

/* =======================
   Word → JSON 抽出
======================= */

func ExtractWordStructure(path string) (*DocTemplate, error) {
    doc, err := document.Open(path)
    if err != nil {
        return nil, err
    }
    defer doc.Close()

    result := &DocTemplate{Type: "word"}
    result.PageSettings = extractPageSettings(doc) // ページ設定を保持

    // 既存の段落・表・画像処理
    var current *Section
    var currentList *Block
    var currentListID int64 = -1

    linkMap, err := buildHyperlinkMapFromXML(path)
    if err != nil {
        log.Printf("警告: ハイパーリンクURL抽出に失敗: %v", err)
    }

    for pi, p := range doc.Paragraphs() {
        text := paragraphPlainText(p)

        if strings.TrimSpace(text) == "" {
            currentList = nil
            currentListID = -1
            if current != nil {
                current.Body = append(current.Body, Block{Kind: "blank_line"})
            }
            continue
        }

        if strings.HasPrefix(p.Style(), "Heading") {
            currentList = nil
            currentListID = -1
            sec := Section{Title: extractParagraphBlock(p, linkMap[pi])}
            result.Sections = append(result.Sections, sec)
            current = &result.Sections[len(result.Sections)-1]
            continue
        }

        if current == nil {
            result.Sections = append(result.Sections, Section{})
            current = &result.Sections[len(result.Sections)-1]
        }

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

            item := extractRuns(p, linkMap[pi])
            currentList.Items = append(currentList.Items, item)
            continue
        }

        currentList = nil
        currentListID = -1
        current.Body = append(current.Body, *extractParagraphBlock(p, linkMap[pi]))
    }

    // 表処理
    for _, tbl := range doc.Tables() {
        if current == nil {
            result.Sections = append(result.Sections, Section{})
            current = &result.Sections[len(result.Sections)-1]
        }
        tableBlock := Block{Kind: "table"}
        for _, row := range tbl.Rows() {
            var rowBlocks []Block
            for _, cell := range row.Cells() {
                for _, p := range cell.Paragraphs() {
                    rowBlocks = append(rowBlocks, Block{
                        Kind: "paragraph",
                        Runs: extractRuns(p, nil),
                    })
                }
            }
            tableBlock.Rows = append(tableBlock.Rows, rowBlocks)
        }
        current.Body = append(current.Body, tableBlock)
    }

    // 画像
    imgCounter := 1
    for _, img := range doc.Images {
        if current == nil {
            result.Sections = append(result.Sections, Section{})
            current = &result.Sections[len(result.Sections)-1]
        }
        data := img.Data()
        if data == nil {
            continue
        }
        imgBlock := Block{
            Kind: "image",
            Image: &ImageBlock{
                Name: fmt.Sprintf("image_%d.png", imgCounter),
                Data: *data,
            },
        }
        current.Body = append(current.Body, imgBlock)
        imgCounter++
    }

    return result, nil
}

// 段落のプレーンテキスト（Runを連結）
func paragraphPlainText(p document.Paragraph) string {
    var b strings.Builder
    for _, r := range p.Runs() {
        b.WriteString(r.Text())
    }
    return b.String()
}

// 段落→Block（Runs抽出含む）
func extractParagraphBlock(p document.Paragraph, linksInPara []xmlHyperlink) *Block {
    return &Block{
        Kind:  "paragraph",
        Style: p.Style(),
        Runs:  extractRuns(p, linksInPara),
    }
}

// Runs抽出：Bold/Italic/Size + 文字列がハイパーリンクアンカーに含まれる場合は URL を付与
func extractRuns(p document.Paragraph, linksInPara []xmlHyperlink) []Run {
    var runs []Run
    for _, r := range p.Runs() {
        text := r.Text()
        run := Run{
            Text:     text,
            Bold:     r.Properties().IsBold(),
            Italic:   r.Properties().IsItalic(),
            FontSize: getFontSize(r),
        }
        // XMLで拾ったハイパーリンクアンカーに該当するならURLを付与
        for _, hl := range linksInPara {
            // 単純一致（必要ならトークン分割や位置合わせを強化）
            if text != "" && strings.Contains(hl.AnchorText, text) {
                run.Hyperlink = hl.URL
                break
            }
        }
        runs = append(runs, run)
    }
    return runs
}

// フォントサイズ：half-points を point に変換
func getFontSize(r document.Run) int {
    props := r.Properties()
    if props.X() == nil || props.X().Sz == nil {
        return 0
    }
    sz := props.X().Sz.ValAttr
    if sz.ST_UnsignedDecimalNumber != nil {
        return int(*sz.ST_UnsignedDecimalNumber / 2)
    }
    return 0
}

/* =======================
   XML直読み：段落インデックスごとのリンクURL辞書
======================= */

// 最低限のXMLモデル
type xmlRelationships struct {
    XMLName      xml.Name          `xml:"Relationships"`
    Relationships []xmlRelationship `xml:"Relationship"`
}
type xmlRelationship struct {
    Id     string `xml:"Id,attr"`
    Type   string `xml:"Type,attr"`
    Target string `xml:"Target,attr"`
}
type xmlDocument struct {
    XMLName xml.Name     `xml:"document"`
    Body    xmlBody      `xml:"body"`
}
type xmlBody struct {
    Paras []xmlParagraph `xml:"p"`
}
type xmlParagraph struct {
    Hyperlinks []xmlHyperlinkNode `xml:"hyperlink"`
}
type xmlHyperlinkNode struct {
    Rid       string            `xml:"id,attr"` // r:id
    RunNodes  []xmlRunNode      `xml:"r"`
}
type xmlRunNode struct {
    Texts []xmlTextNode `xml:"t"`
}
type xmlTextNode struct {
    Text string `xml:",chardata"`
}
type xmlHyperlink struct {
    URL        string
    AnchorText string
}

func buildHyperlinkMapFromXML(docxPath string) (map[int][]xmlHyperlink, error) {
    rc, err := zip.OpenReader(docxPath)
    if err != nil {
        return nil, err
    }
    defer rc.Close()

    var docXML, relsXML []byte
    for _, f := range rc.File {
        switch f.Name {
        case "word/document.xml":
            r, _ := f.Open(); docXML, _ = io.ReadAll(r); r.Close()
        case "word/_rels/document.xml.rels":
            r, _ := f.Open(); relsXML, _ = io.ReadAll(r); r.Close()
        }
    }
    if len(docXML) == 0 || len(relsXML) == 0 {
        return map[int][]xmlHyperlink{}, nil
    }

    // r:id → URL の辞書
    var rels xmlRelationships
    if err := xml.Unmarshal(relsXML, &rels); err != nil {
        return nil, err
    }
    idToURL := map[string]string{}
    for _, r := range rels.Relationships {
        if strings.Contains(strings.ToLower(r.Type), "hyperlink") {
            idToURL[r.Id] = r.Target
        }
    }

    // 段落ごとのハイパーリンク
    var xdoc xmlDocument
    if err := xml.Unmarshal(docXML, &xdoc); err != nil {
        return nil, err
    }
    out := map[int][]xmlHyperlink{}
    for i, p := range xdoc.Body.Paras {
        for _, hl := range p.Hyperlinks {
            url := idToURL[hl.Rid]
            var b strings.Builder
            for _, rn := range hl.RunNodes {
                for _, t := range rn.Texts {
                    b.WriteString(t.Text)
                }
            }
            anchor := b.String()
            if url != "" && anchor != "" {
                out[i] = append(out[i], xmlHyperlink{URL: url, AnchorText: anchor})
            }
        }
    }
    return out, nil
}

/* =======================
   JSON → Word 再構築
======================= */

func ApplyJSONToWordStruct(template *DocTemplate, outputPath string) error {
    doc := document.New()

    // ページ設定をコピー
    if template.PageSettings != nil {
        docSect := doc.X().Body.SectPr
        if docSect == nil {
            docSect = wml.NewCT_SectPr()
            doc.X().Body.SectPr = docSect
        }
        *docSect = *template.PageSettings
    }

    // 箇条書き定義
    numDef := createBulletNumbering(doc)

    for _, sec := range template.Sections {
        if sec.Title != nil {
            p := doc.AddParagraph()
            if sec.Title.Style != "" {
                p.SetStyle(sec.Title.Style)
            }
            applyRuns(p, sec.Title.Runs)
        }
        for _, b := range sec.Body {
            switch b.Kind {
            case "blank_line":
                doc.AddParagraph()
            case "paragraph":
                p := doc.AddParagraph()
                if b.Style != "" {
                    p.SetStyle(b.Style)
                }
                applyRuns(p, b.Runs)
            case "list":
                for _, item := range b.Items {
                    p := doc.AddParagraph()
                    p.SetNumberingDefinition(numDef)
                    p.SetNumberingLevel(b.Indent)
                    applyRuns(p, item)
                }
            case "table":
                tbl := doc.AddTable()
                for _, row := range b.Rows {
                    r := tbl.AddRow()
                    for _, cellBlock := range row {
                        c := r.AddCell()
                        p := c.AddParagraph()
                        if cellBlock.Style != "" {
                            p.SetStyle(cellBlock.Style)
                        }
                        applyRuns(p, cellBlock.Runs)
                    }
                }
            case "image":
                if b.Image != nil {
                    img, err := common.ImageFromBytes(b.Image.Data)
                    if err != nil {
                        return err
                    }
                    imgRef, err := doc.AddImage(img)
                    if err != nil {
                        return err
                    }
                    p := doc.AddParagraph()
                    run := p.AddRun()
                    _, err = run.AddDrawingInline(imgRef)
                    if err != nil {
                        return err
                    }
                }
            }
        }
    }

    return doc.SaveToFile(outputPath)
}
// ハイパーリンク生成は Paragraph.AddHyperLink を使う
func applyRuns(p document.Paragraph, runs []Run) {
    for _, r := range runs {
        if r.Hyperlink != "" {
            hl := p.AddHyperLink()                 // 段落にリンクオブジェクトを追加
            hl.SetTarget(r.Hyperlink)              // URL設定
            hr := hl.AddRun()                      // リンクの中のRunを作成
            hr.AddText(r.Text)                     // テキスト
            hr.Properties().SetBold(r.Bold)
            hr.Properties().SetItalic(r.Italic)
            if r.FontSize > 0 {
                hr.Properties().SetSize(measurement.Distance(r.FontSize * 2))
            }
            continue
        }

        run := p.AddRun()
        run.AddText(r.Text)
        run.Properties().SetBold(r.Bold)
        run.Properties().SetItalic(r.Italic)
        if r.FontSize > 0 {
            run.Properties().SetSize(measurement.Distance(r.FontSize * 2))
        }
    }
}

// 箇条書き定義
func createBulletNumbering(doc *document.Document) document.NumberingDefinition {
    numDef := doc.Numbering.AddDefinition()
    lvl := numDef.AddLevel()
    lvl.SetFormat(wml.ST_NumberFormatBullet)
    lvl.SetText("•")
    lvl.Properties().SetLeftIndent(measurement.Distance(720))
    lvl.Properties().SetHangingIndent(measurement.Distance(360))
    return numDef
}
