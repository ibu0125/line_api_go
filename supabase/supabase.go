package supabase

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
)

type Conversation struct {
	ID string `json:"id"`
}

type User struct {
	ID string `json:"id"`
}



var (
	SUPABASE_URL = os.Getenv("SUPABASE_URL")
	SUPABASE_SERVICE_ROLE_KEY=os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	envErr= errors.New("SUPABASE_URL,SUPABASE_SERVICE_ROLE_KEYが見つかりません")
)

func request(method, p string, body any) (*http.Response, error) {
	if SUPABASE_URL == "" || SUPABASE_SERVICE_ROLE_KEY == "" {
		return nil, envErr
	}

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	// path.Join は使わず、文字列連結で安全に
	fullURL := SUPABASE_URL + p

	req, err := http.NewRequest(method, fullURL, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", SUPABASE_SERVICE_ROLE_KEY)
	req.Header.Set("Authorization", "Bearer "+SUPABASE_SERVICE_ROLE_KEY)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")

	return http.DefaultClient.Do(req)
}


func GetUserByLineID(lineUserID string) (*User, error) {
	resp, err := request(
		"GET",
		"/rest/v1/users?line_user_id=eq."+lineUserID+"&select=id",
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("failed to fetch user")
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	return &users[0], nil
}


func AddMessage(conversationID, role, content string) error {
	_, err := request(
		"POST",
		"/rest/v1/messages",
		map[string]string{
			"conversation_id": conversationID,
			"role":            role,
			"content":         content,
		},
	)
	return err
}


func GetMessages(conversationID string, limit int) ([]map[string]string, error) {
	resp, err := request(
		"GET",
		"/rest/v1/messages?conversation_id=eq."+conversationID+
			"&order=created_at.asc&limit="+strconv.Itoa(limit),
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var msgs []map[string]string
	json.NewDecoder(resp.Body).Decode(&msgs)
	return msgs, nil
}



func GetOrCreateConversation(userID, contextKey string) (string, error) {
	// ① 取得
	resp, err := request(
		"GET",
		"/rest/v1/conversations?user_id=eq."+userID+"&context_key=eq."+contextKey,
		nil,
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var convs []Conversation
	json.NewDecoder(resp.Body).Decode(&convs)

	if len(convs) > 0 {
		return convs[0].ID, nil
	}

	// ② 作成
	var created []Conversation
	resp, err = request(
		"POST",
		"/rest/v1/conversations?select=id",
		map[string]string{
			"user_id":     userID,
			"context_key": contextKey,
		},
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&created)
	return created[0].ID, nil
}



func IsUser(lineUserID string)(bool,error){
	resp,err:=request(
		"GET",
		"/rest/v1/users?line_user_id=eq."+lineUserID,
		nil,
	)

	if  err!=nil {
		return false,err
	}

	defer resp.Body.Close()

	var users []map[string]any

	json.NewDecoder(resp.Body).Decode(&users)

	return len(users)>0,nil

}


func AddUser(lineUserID string)error{
	_,err:=request(
		"POST",
		"/rest/v1/users",
		map[string]string{
			"line_user_id": lineUserID,
		},
	)
	return err
}


func UseAuthCode(code string) (bool, error) {
    resp, err := request(
        "PATCH", 
        "/rest/v1/auth_codes?code=eq."+code+"&used=eq.false",
        map[string]bool{"used": true},
    )
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()
	log.Println("StatusCode:", resp.StatusCode)
    if resp.StatusCode == 204 {
        return true, nil
    }
    return false, nil
}
