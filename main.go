package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
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

// اطلاعات دستی برای اسم انگلیسی و آیکون طلاها
var goldDetails = map[string]struct {
	NameEn string
	Icon   string
}{
	"abshodeh": {"Mithqal Gold", "https://platform.tgju.org/files/images/gold-bar-1622253729.png"},
	"18ayar":   {"18 Karat Gold", "https://platform.tgju.org/files/images/gold-bar-1-1622253841.png"},
	"sekkeh":   {"Imami Coin", "https://platform.tgju.org/files/images/gold-1697963730.png"},
	"bahar":    {"Bahar Azadi Coin", "https://platform.tgju.org/files/images/gold-1-1697963918.png"},
	"nim":      {"Half Coin", "https://platform.tgju.org/files/images/money-1697964123.png"},
	"rob":      {"Quarter Coin", "https://platform.tgju.org/files/images/revenue-1697964369.png"},
	"sek":      {"Gram Coin", "https://platform.tgju.org/files/images/parsian-coin-1697964860.png"},
	"usd_xau":  {"Ounce Gold", "https://platform.tgju.org/files/images/gold-1-1622253769.png"},
}

// Fetch data from API 1 (Currency Prices)
func fetchDataAPI1() (map[string]Currency, error) {
	reqBody := `{"lang": "fa"}`
	req, err := http.NewRequest("POST", api1URL, strings.NewReader(reqBody))
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

		// بررسی کنیم که آرایه price خالی نباشه
		if priceList, ok := item["price"].([]interface{}); ok && len(priceList) > 0 {
			if priceEntry, ok := priceList[0].(map[string]interface{}); ok {
				price = priceEntry["price"].(float64)
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

// Fetch data from Gold API
func fetchGoldData() map[string]Currency {
	reqBody := `{"lang": "fa"}`
	req, err := http.NewRequest("POST", goldURL, strings.NewReader(reqBody))
	if err != nil {
		fmt.Println("Error creating request for Gold API:", err)
		return nil
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching Gold API:", err)
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error decoding Gold API response:", err)
		return nil
	}

	goldItems := result["gold"].([]interface{})
	goldData := make(map[string]Currency)

	for _, item := range goldItems {
		goldMap := item.(map[string]interface{})
		code := goldMap["slug"].(string)

		// گرفتن آخرین قیمت از آرایه قیمت‌ها
		prices := goldMap["price"].([]interface{})
		lastPrice := prices[0].(map[string]interface{})["price"].(float64)

		// تنظیم اطلاعات از `goldDetails`
		details, exists := goldDetails[code]
		enName := ""
		icon := ""
		if exists {
			enName = details.NameEn
			icon = details.Icon
		}

		goldData[code] = Currency{
			Code: code,
			Name: map[string]string{
				"fa": goldMap["name"].(string),
				"en": enName,
			},
			Price: lastPrice,
			Icon:  icon,
		}
	}
	return goldData
}

// Get current time in Jalali format
func getJalaliTime() string {
	loc, _ := time.LoadLocation("Asia/Tehran")
	now := time.Now().In(loc)
	jalaliDate := ptime.New(now)
	return fmt.Sprintf("%04d/%02d/%02d, %02d:%02d",
		jalaliDate.Year(), jalaliDate.Month(), jalaliDate.Day(),
		now.Hour(), now.Minute(),
	)
}

// Process data and save to JSON
func processAndSaveData() error {
	var wg sync.WaitGroup
	var api1Data, goldData map[string]Currency

	wg.Add(2)

	go func() {
		defer wg.Done()
		api1Data = fetchDataAPI1()
	}()

	go func() {
		defer wg.Done()
		goldData = fetchGoldData()
	}()

	wg.Wait()

	finalData := make(map[string]Currency)

	// ترکیب ارزها و طلاها (اگه `nil` نباشن)
	if api1Data != nil {
		for code, data := range api1Data {
			finalData[code] = data
		}
	}
	if goldData != nil {
		for code, data := range goldData {
			finalData[code] = data
		}
	}

	if len(finalData) == 0 {
		fmt.Println("No data available, nothing to save.")
		return nil
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
