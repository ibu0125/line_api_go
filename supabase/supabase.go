package supabase

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
    SUPABASE_URL             = strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
    SUPABASE_SERVICE_ROLE_KEY = os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
    envErr                   = errors.New("SUPABASE_URL または SUPABASE_SERVICE_ROLE_KEY が未設定です")
)

// 共通リクエスト関数
func request(method, p string, body any, prefer string) (*http.Response, error) {
    if SUPABASE_URL == "" || SUPABASE_SERVICE_ROLE_KEY == "" {
        return nil, envErr
    }

    var buf bytes.Buffer
    if body != nil {
        if err := json.NewEncoder(&buf).Encode(body); err != nil {
            return nil, fmt.Errorf("JSONエンコード失敗: %w", err)
        }
    }

    fullURL := SUPABASE_URL + p
    log.Println("Request URL:", fullURL)

    req, err := http.NewRequest(method, fullURL, &buf)
    if err != nil {
        return nil, fmt.Errorf("NewRequest失敗: %w", err)
    }

    // 認証ヘッダ（Service Role Keyを使用）
    req.Header.Set("apikey", SUPABASE_SERVICE_ROLE_KEY)
    req.Header.Set("Authorization", "Bearer "+SUPABASE_SERVICE_ROLE_KEY)

    // コンテンツタイプ/受信タイプ
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    // 戻り値ポリシー（任意）
    if prefer != "" {
        req.Header.Set("Prefer", prefer)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("HTTP送信失敗: %w", err)
    }

    // 失敗時の詳細ログ（400/4xx/5xx）
    if resp.StatusCode >= 400 {
        bodyBytes, _ := io.ReadAll(resp.Body)
        _ = resp.Body.Close()
        log.Printf("HTTP %d エラー応答: %s\n", resp.StatusCode, string(bodyBytes))
        // 再度読み直せるようにボディ復元
        resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
    }

    return resp, nil
}

// ユーザー存在確認
func IsUser(lineUserID string) (bool, error) {
    // 値は URL エンコード
    id := url.QueryEscape(lineUserID)

    // 例: /rest/v1/users?select=*&line_user_id=eq.{id}
    path := fmt.Sprintf("/rest/v1/users?select=*&line_user_id=eq.%s", id)

    resp, err := request("GET", path, nil, "")
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return false, fmt.Errorf("IsUser失敗: status=%d", resp.StatusCode)
    }

    var users []map[string]any
    if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
        return false, fmt.Errorf("JSONデコード失敗: %w", err)
    }

    return len(users) > 0, nil
}

// ユーザー追加
func AddUser(lineUserID string) error {
    payload := map[string]string{
        "line_user_id": lineUserID,
    }

    resp, err := request("POST", "/rest/v1/users", payload, "return=minimal")
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // 成功時 201/204 想定
    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("AddUser失敗: status=%d body=%s", resp.StatusCode, string(body))
    }
    return nil
}

// 認証コード使用（未使用コードのみを used=true に更新）
func UseAuthCode(code string) (bool, error) {
    // 値は URL エンコード
    c := url.QueryEscape(code)

    // & を必ず使う。&amp; は不可。
    // used=is.false フィルタは PostgREST の boolean 演算子
    // PATCH 本文は {"used": true}
    path := fmt.Sprintf("/rest/v1/users?code=eq.%s&used=is.false", c)

    resp, err := request("PATCH", path, map[string]bool{"used": true}, "return=minimal")
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    log.Println("StatusCode:", resp.StatusCode)

    // return=minimal のとき、更新成功は 204 が返る
    if resp.StatusCode == http.StatusNoContent {
        return true, nil
    }

    // 代表的な失敗ケースを補足
    if resp.StatusCode == http.StatusBadRequest {
        return false, errors.New("Bad Request: クエリ式や本文を確認してください（&amp;ではなく&、値のURLエンコード必須）")
    }
    if resp.StatusCode == http.StatusNotFound {
        return false, errors.New("該当する code が見つからない、または既に used=true です")
    }

    // それ以外
    body, _ := io.ReadAll(resp.Body)
    return false, fmt.Errorf("UseAuthCode失敗: status=%d body=%s", resp.StatusCode, string(body))
}
