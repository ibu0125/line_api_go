package supabase

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
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

	// path.Join は使わず、文字列連結で安全に
	fullURL := SUPABASE_URL + p
	log.Println("Request URL:", fullURL)
	log.Println("apikey:",SUPABASE_SERVICE_ROLE_KEY,SUPABASE_URL)

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
