package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
)

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

	u, err := url.Parse(SUPABASE_URL)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, p)

	req, err := http.NewRequest(method, u.String(), &buf) // ← ★ここ
	if err != nil {
		return nil, err
	}

	req.Header.Set("apikey", SUPABASE_SERVICE_ROLE_KEY)
	req.Header.Set("Authorization", "Bearer "+SUPABASE_SERVICE_ROLE_KEY)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")

	return http.DefaultClient.Do(req)
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