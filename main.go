package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/PuerkitoBio/goquery"
    ptime "github.com/yaa110/go-persian-calendar"
)

type Currency struct {
    Code  string  `json:"code"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
    Icon  string  `json:"icon"`
    En    string  `json:"en"`
}

type FinalOutput struct {
    Date       string     `json:"date"`
    Currencies []Currency `json:"currencies"`
}

type Country struct {
    Country string `json:"country"`
    En      string `json:"en"`
}

var currencyMap map[string]string

// 🔑 تبدیل اعداد فارسی به انگلیسی
func faToEnDigits(s string) string {
    replacer := strings.NewReplacer(
        "۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
        "۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
    )
    return replacer.Replace(s)
}

// تبدیل رشته به عدد
func parseNumber(s string) float64 {
    s = faToEnDigits(s)
    s = strings.ReplaceAll(s, "$", "")
    s = strings.ReplaceAll(s, ",", "")
    s = strings.ReplaceAll(s, "٫", ".")
    s = strings.TrimSpace(s)
    f, _ := strconv.ParseFloat(s, 64)
    return f
}

// تغییر متن به Title Case
func toTitleCase(s string) string {
    return strings.Title(strings.ToLower(strings.TrimSpace(s)))
}

// کریپتو
func fetchCryptoData() ([]Currency, error) {
    resp, err := http.Get("https://alanchand.com/crypto-price")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return nil, err
    }

    var cryptoData []Currency
    doc.Find("table.cryptoTbl tbody tr").Each(func(i int, s *goquery.Selection) {
        code := strings.ToLower(strings.TrimSpace(s.Find(".symbolCurr").Text()))
        nameFa := strings.TrimSpace(s.Find(".faCurr").Text())
        nameEn := strings.TrimSpace(s.Find(".enCurr").Text())
        tomanStr := s.Find(".tmn").Text()
        dollarStr := s.Find(".dlr").Text()
        icon, _ := s.Find(".CurrIco").Attr("src")
        if !strings.HasPrefix(icon, "http") {
            icon = "https://alanchand.com" + icon
        }

        toman := parseNumber(tomanStr)
        dollar := parseNumber(dollarStr)

        price := dollar
        if code == "usdt" || code == "dai" {
            price = toman
        }
        if code == "btc" {
            priceStr := fmt.Sprintf("%.0f", price)
            price, _ = strconv.ParseFloat(priceStr, 64)
        }

        cryptoData = append(cryptoData, Currency{
            Code:  code,
            Name:  nameFa,
            Price: price,
            Icon:  icon,
            En:    toTitleCase(nameEn),
        })
    })
    return cryptoData, nil
}

// ارزها
func fetchDataCurrency(cm map[string]string) ([]Currency, error) {
    resp, err := http.Get("https://alanchand.com/currencies-price")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return nil, err
    }

    var data []Currency
    doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
        codeAttr, _ := s.Attr("onclick")
        code := strings.TrimPrefix(strings.TrimSuffix(codeAttr, "'"), "window.location='/currencies-price/")
        nameFa := strings.TrimSpace(s.Find("td.currName").Text())
        priceStr := strings.TrimSpace(s.Find("td.sellPrice").Contents().First().Text())

        price := parseNumber(priceStr)

        if code != "" {
            flag := code
            enName := cm[flag]
            if enName == "" {
                enName = toTitleCase(code)
            }
            // اصلاح آیکون برای یورو
            if flag == "eu" {
                flag = "european_union"
            }
            icon := fmt.Sprintf("https://raw.githubusercontent.com/HatScripts/circle-flags/refs/heads/gh-pages/flags/%s.svg", flag)

            data = append(data, Currency{
                Code:  code,
                Name:  nameFa,
                Price: price,
                Icon:  icon,
                En:    enName,
            })
        }
    })
    return data, nil
}

// طلا
func fetchGoldData() ([]Currency, error) {
    // مپ آیکون‌ها
    var goldIcons = map[string]string{
        "abshodeh": "https://platform.tgju.org/files/images/gold-bar-1622253729.png",
        "18ayar": "https://platform.tgju.org/files/images/gold-bar-1-1622253841.png",
        "sekkeh": "https://platform.tgju.org/files/images/gold-1697963730.png",
        "bahar": "https://platform.tgju.org/files/images/gold-1-1697963918.png",
        "nim": "https://platform.tgju.org/files/images/money-1697964123.png",
        "rob": "https://platform.tgju.org/files/images/revenue-1697964369.png",
        "sek": "https://platform.tgju.org/files/images/parsian-coin-1697964860.png",
        "usd_xau": "https://platform.tgju.org/files/images/gold-1-1622253769.png",
        "xag": "https://platform.tgju.org/files/images/silver-1624079710.png"
    }

    resp, err := http.Get("https://alanchand.com/gold-price")
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    doc, err := goquery.NewDocumentFromReader(resp.Body)
    if err != nil {
        return nil, err
    }

    var data []Currency
    doc.Find("table tbody tr").Each(func(i int, s *goquery.Selection) {
        codeAttr, _ := s.Attr("onclick")
        code := strings.TrimPrefix(strings.TrimSuffix(codeAttr, "'"), "window.location='/gold-price/")
        nameFa := strings.TrimSpace(s.Find("td").First().Text())
        priceStr := strings.TrimSpace(s.Find("td.priceTd").First().Contents().First().Text())

        price := parseNumber(priceStr)

        icon := ""
        if val, exists := goldIcons[code]; exists {
            icon = val
        }

        if code != "" {
            codeName := code
            if codeName == "sek" {
                codeName = "gram"
            }

            data = append(data, Currency{
                Code:  codeName,
                Name:  nameFa,
                Price: price,
                Icon:  icon,
                En:    toTitleCase(code),
            })
        }
    })
    return data, nil
}

// تاریخ جلالی
func getJalaliTime() string {
    loc, _ := time.LoadLocation("Asia/Tehran")
    now := time.Now().In(loc)
    jalali := ptime.New(now)
    return fmt.Sprintf("%04d/%02d/%02d, %02d:%02d",
        jalali.Year(), jalali.Month(), jalali.Day(),
        now.Hour(), now.Minute())
}

func loadCurrencyMap() error {
    data, err := os.ReadFile("currencies.json")
    if err != nil {
        return err
    }

    var countries []Country
    err = json.Unmarshal(data, &countries)
    if err != nil {
        return err
    }

    currencyMap = make(map[string]string)
    for _, c := range countries {
        currencyMap[c.Country] = c.En
    }
    return nil
}

func main() {
    err := loadCurrencyMap()
    if err != nil {
        fmt.Println("Error loading currencies.json:", err)
        return
    }

    var wg sync.WaitGroup
    var currencies, golds, cryptos []Currency
    var err1, err2, err3 error

    wg.Add(3)
    go func() {
        defer wg.Done()
        currencies, err1 = fetchDataCurrency(currencyMap)
    }()
    go func() {
        defer wg.Done()
        golds, err2 = fetchGoldData()
    }()
    go func() {
        defer wg.Done()
        cryptos, err3 = fetchCryptoData()
    }()
    wg.Wait()

    if err1 != nil {
        fmt.Println("Error currency:", err1)
    }
    if err2 != nil {
        fmt.Println("Error gold:", err2)
    }
    if err3 != nil {
        fmt.Println("Error crypto:", err3)
    }

    finalData := append(append(currencies, golds...), cryptos...)

    output := FinalOutput{
        Date:       getJalaliTime(),
        Currencies: finalData,
    }

    jsonData, _ := json.MarshalIndent(output, "", "  ")
    _ = os.WriteFile("arz.json", jsonData, 0644)
    fmt.Println("✅ arz.json ساخته شد")
}