package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type MarsHydroAPI struct {
	Email         string
	Password      string
	WifiName      string
	Token         string
	Timezone 	string
	Language 	string
	BaseURL       string
	mu            sync.Mutex
	LastLoginTime time.Time
	LoginInterval time.Duration
	DeviceID      string
	GroupID       string
}

func NewMarsHydroAPI(email, password, WifiName, Timezone, Language string) *MarsHydroAPI {
	return &MarsHydroAPI{
		Email:         email,
		Password:      password,
		WifiName:      WifiName,
		Timezone:      Timezone,
		Language: 	Language,
		BaseURL:       "https://api.lgledsolutions.com/api/android",
		LoginInterval: 300 * time.Second,
	}
}

func (api *MarsHydroAPI) generateSystemData() string {
	data := map[string]interface{}{
		"reqId":      time.Now().UnixNano() / 1e6,
		"appVersion": "1.2.0",
		"osType":     "android",
		"osVersion":  "14",
		"deviceType": "SM-S928C",
		"deviceId":   api.DeviceID,
		"netType":    "wifi",
		"wifiName":   api.WifiName,
		"timestamp":  time.Now().Unix(),
		"token":      api.Token,
		"timezone":  api.Timezone,
		"language":  	api.Language,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func (api *MarsHydroAPI) Login() error {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.Token != "" && time.Since(api.LastLoginTime) < api.LoginInterval {
		log.Println("Token still valid, skipping login.")
		return nil
	}

	systemData := api.generateSystemData()
	payload := map[string]interface{}{
		"email":       api.Email,
		"password":    api.Password,
		"loginMethod": "1",
	}
	bPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", api.BaseURL+"/ulogin/mailLogin/v1", bytes.NewBuffer(bPayload))
	if err != nil {
		return err
	}
	req.Header.Set("systemData", systemData)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return err
	}
	// ...process response and logging...
	data, ok := resData["data"].(map[string]interface{})
	if !ok {
		return errors.New("invalid login response")
	}
	token, ok := data["token"].(string)
	if !ok {
		return errors.New("token not found in response")
	}
	api.Token = token
	api.LastLoginTime = time.Now()
	log.Println("Login successful, token received.")
	return nil
}

func (api *MarsHydroAPI) ToggleSwitch(isClose bool, deviceID string) (map[string]interface{}, error) {
	if err := api.ensureToken(); err != nil {
		return nil, err
	}

	systemData := api.generateSystemData()
	payload := map[string]interface{}{
		"isClose":  isClose,
		"deviceId": deviceID,
		"groupId":  nil,
	}
	bPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", api.BaseURL+"/udm/lampSwitch/v1", bytes.NewBuffer(bPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("systemData", systemData)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return nil, err
	}

	// If token expired (code 102), re-authenticate.
	if code, ok := resData["code"].(string); ok && code == "102" {
		log.Println("Token expired, re-authenticating...")
		if err := api.Login(); err != nil {
			return nil, err
		}
		return api.ToggleSwitch(isClose, deviceID)
	}
	return resData, nil
}

func (api *MarsHydroAPI) GetLightData() (map[string]interface{}, error) {
	if err := api.ensureToken(); err != nil {
		return nil, err
	}

	systemData := api.generateSystemData()
	payload := map[string]interface{}{
		"currentPage": 0,
		"type":        nil,
		"productType": "LIGHT",
	}
	bPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", api.BaseURL+"/udm/getDeviceList/v1", bytes.NewBuffer(bPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("systemData", systemData)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Host", "api.lgledsolutions.com")
	req.Header.Set("User-Agent", "Python/3.x")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return nil, err
	}

	if code, ok := resData["code"].(string); !ok || code != "000" {
		log.Println("Error in API response:", resData["msg"])
		return nil, errors.New("error retrieving light devices")
	}

	data, _ := resData["data"].(map[string]interface{})
	list, ok := data["list"].([]interface{})
	if !ok || len(list) == 0 {
		log.Println("No light devices found.")
		return nil, errors.New("no light devices available")
	}
	deviceData, ok := list[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid device data")
	}
	// Try retrieving device id as string; if not, check if it's numeric.
	if id, exists := deviceData["id"]; exists {
		switch v := id.(type) {
		case string:
			if v != "" {
				api.DeviceID = v
			}
		case float64:
			api.DeviceID = fmt.Sprintf("%.0f", v)
		}
	}
	// Fallback: check "deviceId" field similarly.
	if api.DeviceID == "" {
		if alt, exists := deviceData["deviceId"]; exists {
			switch v := alt.(type) {
			case string:
				if v != "" {
					api.DeviceID = v
				}
			case float64:
				api.DeviceID = fmt.Sprintf("%.0f", v)
			}
		}
	}
	if api.DeviceID == "" {
		return nil, errors.New("device id not found in response")
	}
	// Retrieve group id if available.
	if gid, exists := deviceData["groupId"]; exists {
		switch v := gid.(type) {
		case string:
			api.GroupID = v
		case float64:
			api.GroupID = fmt.Sprintf("%.0f", v)
		}
	} else {
		api.GroupID = ""
	}

	lightData := map[string]interface{}{
		"deviceName":      deviceData["deviceName"],
		"deviceLightRate": deviceData["deviceLightRate"],
		"isClose":         deviceData["isClose"],
		"id":              api.DeviceID,
		"deviceImage":     deviceData["deviceImg"],
		"groupId":         api.GroupID,
	}
	return lightData, nil
}

func (api *MarsHydroAPI) SetBrightness(brightness interface{}) (error) {
	if api.DeviceID == "" {
		if _, err := api.GetLightData(); err != nil {
			return err
		}
	}

	if err := api.ensureToken(); err != nil {
		return err
	}

	systemData := api.generateSystemData()
	payload := map[string]interface{}{
		"light":    brightness,
		"deviceId": api.DeviceID,
		"groupId":  api.GroupID, // use groupId (may be empty)
	}
	bPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", api.BaseURL+"/udm/adjustLight/v1", bytes.NewBuffer(bPayload))
	if err != nil {
		return err
	}
	req.Header.Set("systemData", systemData)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Host", "api.lgledsolutions.com")
	req.Header.Set("User-Agent", "Python/3.x") // not checked but mehh

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var resData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&resData); err != nil {
		return err
	}

	// SUCCESS WOULD BE -> SetBrightness response: map[code:000 data:map[commandMap:map[] current:0 nodeDeviceId:<nil>] msg:success subCode:<nil>]
	if code, ok := resData["code"].(string); !ok || code != "000" {
		log.Println("Error in API response:", resData["msg"])
		return errors.New("received error response")
	}

	log.Println("Brightness set successfully.")

	return nil
}

func (api *MarsHydroAPI) ensureToken() error {
	if api.Token == "" {
		return api.Login()
	}
	return nil
}
