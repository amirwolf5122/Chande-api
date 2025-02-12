package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	ptime "github.com/yaa110/go-persian-calendar"
)

const (
	api1URL = "https://admin.alanchand.com/api/arz"
	goldURL = "https://admin.alanchand.com/api/gold"
)

// Currency struct for storing price details
type Currency struct {
	Code  string            `json:"code"`
	Name  map[string]string `json:"name"`
	Price float64           `json:"price"`
	Icon  string            `json:"icon"`
}

// Final output struct with update time
type FinalOutput struct {
	Date       string              `json:"date"`
	Currencies map[string]Currency `json:"currencies"`
}

// Fetch data from API 1
func fetchDataAPI1() (map[string]Currency, error) {
	reqBody := `{"lang": "fa"}`
	req, err := http.NewRequest("POST", api1URL, ioutil.NopCloser(strings.NewReader(reqBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string][]map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	currencies := make(map[string]Currency)
	for _, item := range result["arz"] {
		code := item["slug"].(string)
		price := 0.0
		if prices, ok := item["price"].([]interface{}); ok && len(prices) > 0 {
			if priceMap, ok := prices[0].(map[string]interface{}); ok {
				price = priceMap["price"].(float64)
			}
		}
		currencies[code] = Currency{
			Code: code,
			Name: map[string]string{
				"fa": item["name"].(string),
			},
			Price: price,
			Icon:  fmt.Sprintf("https://raw.githubusercontent.com/hampusborgos/country-flags/main/svg/%s.svg", item["flag"].(string)),
		}
	}
	return currencies, nil
}

// Fetch gold data from the new API
func fetchGoldData() (map[string]Currency, error) {
	reqBody := `{"lang": "fa"}`
	req, err := http.NewRequest("POST", goldURL, ioutil.NopCloser(strings.NewReader(reqBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Manual settings for gold icons and names
	goldNames := map[string]map[string]string{
		"abshodeh": {"fa": "طلای آبشده", "en": "Mithqal Gold"},
		"18ayar":   {"fa": "طلای 18 عیار", "en": "18 Karat Gold"},
		"sekkeh":   {"fa": "سکه امامی", "en": "Emami Coin"},
		"bahar":    {"fa": "سکه بهار آزادی", "en": "Bahar Azadi Coin"},
		"nim":      {"fa": "نیم سکه", "en": "Half Coin"},
		"rob":      {"fa": "ربع سکه", "en": "Quarter Coin"},
		"sek":      {"fa": "سکه گرمی", "en": "Gram Coin"},
		"usd_xau":  {"fa": "انس طلا", "en": "Ounce Gold"},
	}

	goldIcons := map[string]string{
		"abshodeh": "https://platform.tgju.org/files/images/gold-bar-1622253729.png",
		"18ayar":   "https://platform.tgju.org/files/images/gold-bar-1-1622253841.png",
		"sekkeh":   "https://platform.tgju.org/files/images/gold-1697963730.png",
		"bahar":    "https://platform.tgju.org/files/images/gold-1-1697963918.png",
		"nim":      "https://platform.tgju.org/files/images/money-1697964123.png",
		"rob":      "https://platform.tgju.org/files/images/revenue-1697964369.png",
		"sek":      "https://platform.tgju.org/files/images/parsian-coin-1697964860.png",
		"usd_xau":  "https://platform.tgju.org/files/images/gold-1-1622253769.png",
	}

	goldCurrencies := make(map[string]Currency)
	for _, item := range result["gold"].([]interface{}) {
		code := item.(map[string]interface{})["slug"].(string)
		price := 0.0
		if prices, ok := item.(map[string]interface{})["price"].([]interface{}); ok && len(prices) > 0 {
			if priceMap, ok := prices[0].(map[string]interface{}); ok {
				price = priceMap["price"].(float64)
			}
		}
		goldCurrencies[code] = Currency{
			Code:  code,
			Name:  goldNames[code],
			Price: price,
			Icon:  goldIcons[code],
		}
	}
	return goldCurrencies, nil
}

// Get current time in Jalali format
func getJalaliTime() string {
	loc, _ := time.LoadLocation("Asia/Tehran") // Set timezone to Tehran
	now := time.Now().In(loc)                  // Convert current time to Tehran timezone
	jalaliDate := ptime.New(now)
	return fmt.Sprintf("%04d/%02d/%02d, %02d:%02d", jalaliDate.Year(), jalaliDate.Month(), jalaliDate.Day(), now.Hour(), now.Minute())
}

// Process data and save to JSON
func processAndSaveData() error {
	api1Data, err1 := fetchDataAPI1()
	goldData, err2 := fetchGoldData()
	if err1 != nil {
		fmt.Println("Error fetching data from API 1:", err1)
	}
	if err2 != nil {
		fmt.Println("Error fetching gold data:", err2)
	}

	finalData := make(map[string]Currency)

	// Add currencies from API 1
	for code, data := range api1Data {
		finalData[code] = data
	}

	// Add gold data
	for code, data := range goldData {
		finalData[code] = data
	}

	// Create final output with timestamp
	output := FinalOutput{
		Date:       getJalaliTime(),
		Currencies: finalData,
	}

	// Save to JSON file
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("arz.json", jsonData, 0644)
}

func main() {
	if err := processAndSaveData(); err != nil {
		fmt.Println("Error saving data:", err)
	} else {
		fmt.Println("Data successfully saved.")
	}
}
