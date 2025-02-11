package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	ptime "github.com/yaa110/go-persian-calendar" // تغییر نام ایمپورت به ptime
)

const (
	api1URL = "https://admin.alanchand.com/api/arz"
	api2URL = "https://btime.mastkhiar.xyz/v2/arz/"
)

var cryptoList = map[string]bool{
	"Bitcoin": true, "Ethereum": true, "Dogecoin": true, "Binance Coin": true,
}

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
		if len(item["price"].([]interface{})) > 0 {
			price = item["price"].([]interface{})[0].(map[string]interface{})["price"].(float64)
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

// Fetch data from API 2
func fetchDataAPI2() (map[string]Currency, error) {
	currenciesList := []string{
		"USD", "EUR", "SEKEE", "GRAM", "MITHQAL", "AZADI", "THB",
		"DKK", "BRL", "RUB", "INR", "CNY", "CHF", "AUD", "GBP",
		"TRY", "JPY", "BTC", "ETH", "USDT", "DOGE", "BNB", "CAD",
	}

	var wg sync.WaitGroup
	dataChan := make(chan Currency, len(currenciesList))

	for _, code := range currenciesList {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			resp, err := http.Get(api2URL + code)
			if err != nil || resp.StatusCode != 200 {
				return
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return
			}

			priceKey := "rialPrice"
			if cryptoList[code] {
				priceKey = "price"
			}

			dataChan <- Currency{
				Code: code,
				Name: map[string]string{
					"fa": result["name"].(map[string]interface{})["fa"].(string),
					"en": result["name"].(map[string]interface{})["en"].(string),
				},
				Price: result[priceKey].(float64),
				Icon:  result["icon"].(string),
			}
		}(code)
	}

	wg.Wait()
	close(dataChan)

	currencies := make(map[string]Currency)
	for currency := range dataChan {
		currencies[currency.Code] = currency
	}
	return currencies, nil
}

// Get current time in Jalali format
func getJalaliTime() string {
	now := time.Now()
	jalaliDate := ptime.New(now) // استفاده از ptime به جای persian
	return fmt.Sprintf("%04d/%02d/%02d - %02d:%02d", jalaliDate.Year(), jalaliDate.Month(), jalaliDate.Day(), now.Hour(), now.Minute())
}

// Process data and save to JSON
func processAndSaveData() error {
	api1Data, err1 := fetchDataAPI1()
	api2Data, err2 := fetchDataAPI2()
	if err1 != nil {
		fmt.Println("Error fetching data from API 1:", err1)
	}
	if err2 != nil {
		fmt.Println("Error fetching data from API 2:", err2)
	}

	finalData := make(map[string]Currency)

	// Compare and update prices
	for code, data1 := range api1Data {
		if data2, exists := api2Data[code]; exists {
			if data2.Price > data1.Price {
				finalData[code] = data2
			} else {
				finalData[code] = data1
			}
		} else {
			finalData[code] = data1
		}
	}

	// Add missing currencies from API 2
	for code, data2 := range api2Data {
		if _, exists := api1Data[code]; !exists {
			finalData[code] = data2
		}
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

	return ioutil.WriteFile("currency_data.json", jsonData, 0644)
}

func main() {
	if err := processAndSaveData(); err != nil {
		fmt.Println("Error saving data:", err)
	} else {
		fmt.Println("Data successfully saved.")
	}
}