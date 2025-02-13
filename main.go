package main

import (
	"bytes"
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
	api1URL  = "https://admin.alanchand.com/api/arz"
	goldURL  = "https://admin.alanchand.com/api/gold"
)

// Currency struct for storing price details
type Currency struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Icon  string  `json:"icon"`
	En    string  `json:"en"` // اضافه شده
}

// Final output struct with update time
type FinalOutput struct {
	Date       string    `json:"date"`
	Currencies []Currency `json:"currencies"` // تغییر به آرایه
}

// اطلاعات دستی برای آیکون طلاها و اطلاعات اضافی
var goldDetails = map[string]struct {
	Icon string
	Name string
	En   string
}{
	"abshodeh": {"https://platform.tgju.org/files/images/gold-bar-1622253729.png", "مثقال طلا", "Gold Mithqal"},
	"18ayar":   {"https://platform.tgju.org/files/images/gold-bar-1-1622253841.png", "طلا 18 عیار", "18 Karat Gold"},
	"sekkeh":   {"https://platform.tgju.org/files/images/gold-1697963730.png", "سکه امامی", "Emami Coin"},
	"bahar":    {"https://platform.tgju.org/files/images/gold-1-1697963918.png", "سکه بهار آزادی", "Bahar Azadi Coin"},
	"nim":      {"https://platform.tgju.org/files/images/money-1697964123.png", "نیم سکه", "Half Coin"},
	"rob":      {"https://platform.tgju.org/files/images/revenue-1697964369.png", "ربع سکه", "Quarter Coin"},
	"sek":      {"https://platform.tgju.org/files/images/parsian-coin-1697964860.png", "سکه گرمی", "Gram Coin"},
	"usd_xau":  {"https://platform.tgju.org/files/images/gold-1-1622253769.png", "انس طلا", "USD Gold"},
}

// Fetch data from API 1 (Currency Prices)
func fetchDataAPI1() ([]Currency, error) {
	enMap, err := loadEnData()
	if err != nil {
		return nil, err
	}

	data := map[string]string{"lang": "fa"}
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", api1URL, bytes.NewReader(body))
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

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch data: %v", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	var currencies []Currency
	if arzData, ok := result["arz"].([]interface{}); ok {
		for _, item := range arzData {
			itemData := item.(map[string]interface{})
			code := itemData["slug"].(string)
			name := itemData["name"].(string)
			icon := fmt.Sprintf("https://raw.githubusercontent.com/hampusborgos/country-flags/main/svg/%s.svg", itemData["flag"].(string))

			// اگر اسم کشور "-" داشت، از لیست حذفش می‌کنیم
			if strings.Contains(code, "-") {
				continue
			}

			var price float64
			if prices, ok := itemData["price"].([]interface{}); ok && len(prices) > 0 {
				price = prices[0].(map[string]interface{})["price"].(float64)
			}

			// اضافه کردن en از فایل
			en, exists := enMap[itemData["flag"].(string)] // استفاده از flag به جای code
			if !exists {
				en = name // اگر در فایل نبود، اسم رو قرار می‌دهیم
			}

			currencies = append(currencies, Currency{
				Code:  code,
				Name:  name,
				Price: price,
				Icon:  icon,
				En:    en,
			})
		}
	}

	return currencies, nil
}

// Fetch data from Gold API
func fetchGoldData() ([]Currency, error) {
	reqBody := `{"lang": "fa"}`
	req, err := http.NewRequest("POST", goldURL, strings.NewReader(reqBody))
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

	goldItems := result["gold"].([]interface{})
	var goldData []Currency

	for _, item := range goldItems {
		goldMap := item.(map[string]interface{})
		code := goldMap["slug"].(string)

		prices := goldMap["price"].([]interface{})
		lastPrice := 0.0
		if len(prices) > 0 {
			lastPrice = prices[0].(map[string]interface{})["price"].(float64)
		}

		// اگر کد sek بود، آن را به gram تغییر بده (بخاطر همنام بودن ی ارز دیگه)
		if code == "sek" {
			code = "gram"
		}

		// گرفتن اطلاعات از `goldDetails`
		details, exists := goldDetails[code]
		icon := ""
		name := ""
		en := ""
		if exists {
			icon = details.Icon
			name = details.Name
			en = details.En
		}

		goldData = append(goldData, Currency{
			Code:  code,
			Name:  name,
			Price: lastPrice,
			Icon:  icon,
			En:    en,
		})
	}
	return goldData, nil
}
// Get current time in Jalali format
func getJalaliTime() string {
	loc, _ := time.LoadLocation("Asia/Tehran")
	now := time.Now().In(loc)
	jalaliDate := ptime.New(now)
	return fmt.Sprintf("%04d/%02d/%02d, %02d:%02d", jalaliDate.Year(), jalaliDate.Month(), jalaliDate.Day(), now.Hour(), now.Minute())
}

// Load the English names from the currencies.json file
func loadEnData() (map[string]string, error) {
	var enData []struct {
		Country string `json:"country"`
		En      string `json:"en"`
	}

	data, err := ioutil.ReadFile("currencies.json")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &enData)
	if err != nil {
		return nil, err
	}

	enMap := make(map[string]string)
	for _, item := range enData {
		enMap[item.Country] = item.En
	}

	return enMap, nil
}

// Process data and save to JSON
func processAndSaveData() error {
	var wg sync.WaitGroup
	var api1Data, goldData []Currency
	var err1, errGold error

	wg.Add(2)

	go func() {
		defer wg.Done()
		api1Data, err1 = fetchDataAPI1()
	}()

	go func() {
		defer wg.Done()
		goldData, errGold = fetchGoldData()
	}()

	wg.Wait()

	if err1 != nil {
		fmt.Println("Error fetching data from API 1:", err1)
	}
	if errGold != nil {
		fmt.Println("Error fetching gold data:", errGold)
	}

	// ترکیب ارزها و طلاها
	var finalData []Currency
	finalData = append(finalData, api1Data...)
	finalData = append(finalData, goldData...)

	// ایجاد خروجی نهایی
	output := FinalOutput{
		Date:       getJalaliTime(),
		Currencies: finalData,
	}

	// ذخیره در فایل JSON
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
