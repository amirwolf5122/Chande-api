package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"regexp"
	"sync"
	"time"

	ptime "github.com/yaa110/go-persian-calendar"
)

const (
	currencyURL  = "https://admin.alanchand.com/api/arz"
	goldURL  = "https://admin.alanchand.com/api/gold"
	cryptoURL = "https://admin.alanchand.com/api/crypto"
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

// تبدیل متن به Title Case
func toTitleCase(s string) string {
	re := regexp.MustCompile(`\b[a-z]\w*\b`)
	return re.ReplaceAllStringFunc(s, func(word string) string {
		return strings.ToUpper(string(word[0])) + word[1:]
	})
}


// اطلاعات دستی برای آیکون طلاها و اطلاعات اضافی
var goldDetails = map[string]struct {
	Icon string
	Name string
	En   string
}{
	"abshodeh": {"https://platform.tgju.org/files/images/gold-bar-1622253729.png", "مثقال طلا", "Mithqal"},
	"18ayar":   {"https://platform.tgju.org/files/images/gold-bar-1-1622253841.png", "طلا 18 عیار", "18 Karat"},
	"sekkeh":   {"https://platform.tgju.org/files/images/gold-1697963730.png", "سکه امامی", "Emami"},
	"bahar":    {"https://platform.tgju.org/files/images/gold-1-1697963918.png", "سکه بهار آزادی", "Bahar Azadi"},
	"nim":      {"https://platform.tgju.org/files/images/money-1697964123.png", "نیم سکه", "Half Coin"},
	"rob":      {"https://platform.tgju.org/files/images/revenue-1697964369.png", "ربع سکه", "Quarter Coin"},
	"sek":      {"https://platform.tgju.org/files/images/parsian-coin-1697964860.png", "سکه گرمی", "Gram Coin"},
	"usd_xau":  {"https://platform.tgju.org/files/images/gold-1-1622253769.png", "انس طلا", "USD Gold"},
}

// Fetch data from Currency API
func fetchDataCurrency() ([]Currency, error) {
	enMap, err := loadEnData()
	if err != nil {
		return nil, err
	}

	data := map[string]string{"lang": "fa"}
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", currencyURL, bytes.NewReader(body))
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
			iconFlag := itemData["flag"].(string)
			if iconFlag == "eu" {
				iconFlag = "european_union"
			}
			icon := fmt.Sprintf("https://raw.githubusercontent.com/HatScripts/circle-flags/refs/heads/gh-pages/flags/%s.svg", iconFlag)

			// حذف دلار هرات و ...
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
			
			en = toTitleCase(en)
			
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

		// گرفتن اطلاعات از `goldDetails`
		details, exists := goldDetails[code]
		icon := ""
		name := ""
		en := ""
		if exists {
			icon = details.Icon
			name = details.Name
			en = toTitleCase(details.En)
		}
		
		// اگر کد sek بود، آن را به gram تغییر بده (بخاطر همنام بودن ی ارز دیگه)
		if code == "sek" {
			code = "gram"
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

// Fetch data from Crypto API
func fetchCryptoData() ([]Currency, error) {
	reqBody := `{}`
	req, err := http.NewRequest("POST", cryptoURL, strings.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cryptoItems []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&cryptoItems); err != nil {
		return nil, err
	}

	var cryptoData []Currency
	for _, item := range cryptoItems {
		code := item["slug"].(string)
		name := item["fname"].(string)
		price := item["price"].(float64)  // قیمت دلار
		toman := item["toman"].(float64) // قیمت تومان
		en := toTitleCase(item["name"].(string))

		if code == "usdt" || code == "dai" {
			price = toman
		}

		icon := fmt.Sprintf("https://alanchand.com/images/logo/crypto/%s.svg", strings.ToLower(code))

		cryptoData = append(cryptoData, Currency{
			Code:  code,
			Name:  name,
			Price: price,  // قیمت دلاری (به جز تتر که تومان هست)
			Icon:  icon,
			En:    en,
		})
	}

	return cryptoData, nil
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
	var currencyData, goldData, cryptoData []Currency
	var err1, errGold, errcrypto error

	wg.Add(3)

	go func() {
		defer wg.Done()
		currencyData, err1 = fetchDataCurrency()
	}()

	go func() {
		defer wg.Done()
		goldData, errGold = fetchGoldData()
	}()
	
	go func() {
		defer wg.Done()
		cryptoData, errcrypto = fetchCryptoData()
	}()
	
	wg.Wait()

	if err1 != nil {
		fmt.Println("Error fetching data from API 1:", err1)
	}
	if errGold != nil {
		fmt.Println("Error fetching gold data:", errGold)
	}
	if errcrypto != nil {
		fmt.Println("Error fetching crypto data:", errcrypto)
	}
	
	// ترکیب ارزها
	var finalData []Currency
	finalData = append(finalData, currencyData...)
	finalData = append(finalData, goldData...)
	finalData = append(finalData, cryptoData...)

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
