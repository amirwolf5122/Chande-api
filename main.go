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
	"sekkeh":   {"Emami Coin", "https://platform.tgju.org/files/images/gold-1697963730.png"},
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

	// خواندن کل پاسخ API
	body, _ := ioutil.ReadAll(resp.Body)

	// `result` باید `map[string]interface{}` باشه، چون خروجی API یه `map` است
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	currencies := make(map[string]Currency)
	for _, value := range result { // اینجا `key` رو حذف کردم
		item, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		code, _ := item["slug"].(string)
		name, _ := item["name"].(string)
		flag, _ := item["flag"].(string)

		price := 0.0
		if priceList, ok := item["price"].([]interface{}); ok && len(priceList) > 0 {
			if priceEntry, ok := priceList[0].(map[string]interface{}); ok {
				if p, exists := priceEntry["price"].(float64); exists {
					price = p
				}
			}
		}

		currencies[code] = Currency{
			Code:  code,
			Name:  map[string]string{"fa": name},
			Price: price,
			Icon:  fmt.Sprintf("https://raw.githubusercontent.com/hampusborgos/country-flags/main/svg/%s.svg", flag),
		}
	}

	return currencies, nil
}

// Fetch data from Gold API
func fetchGoldData() (map[string]Currency, error) {
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
	goldData := make(map[string]Currency)

	for _, item := range goldItems {
		goldMap := item.(map[string]interface{})
		code := goldMap["slug"].(string)

		// گرفتن آخرین قیمت از آرایه قیمت‌ها
		prices := goldMap["price"].([]interface{})
		lastPrice := 0.0
		if len(prices) > 0 {
			lastPrice = prices[0].(map[string]interface{})["price"].(float64)
		}

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
	return goldData, nil
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

	finalData := make(map[string]Currency)

	// بررسی می‌کنیم که `api1Data` مقدار داشته باشه
	if api1Data != nil {
		for code, data := range api1Data {
			finalData[code] = data
		}
	}

	// بررسی می‌کنیم که `goldData` مقدار داشته باشه
	if goldData != nil {
		for code, data := range goldData {
			finalData[code] = data
		}
	}

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
