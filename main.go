package main

import (
    "encoding/json"
    "fmt"
    "html/template"
    "log"          
    "net/http"
    "net/url"
    "strings"      
    "time"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

type CardImages struct {
	Small string `json:"small"`
	Large string `json:"large"`
}

type CardSet struct {
	Name   string `json:"name"`
	Series string `json:"series"`
}

type CardPrices struct {
	AverageSellPrice float64 `json:"averageSellPrice"`
	TrendPrice       float64 `json:"trendPrice"`
}

type CardMarket struct {
	URL    string     `json:"url"`
	Prices CardPrices `json:"prices"`
}

type Card struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	HP         string     `json:"hp"`
	Rarity     string     `json:"rarity"`
	Types      []string   `json:"types"`
	Subtypes   []string   `json:"subtypes"`
	Set        CardSet    `json:"set"`
	Images     CardImages `json:"images"`
	Cardmarket CardMarket `json:"cardmarket"`
	Number     string     `json:"number"`
	Artist     string     `json:"artist"`
}

type APIResponse struct {
	Data []Card `json:"data"`
}

type SingleCardResponse struct {
	Data Card `json:"data"`
}

type SearchPageData struct {
	Query string
	Cards []Card
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/about", aboutHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/card/{id}", cardHandler)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "home.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "About this app")
	fmt.Println("Method:", r.Method)
	fmt.Println("URL:", r.URL.Path)
	fmt.Println("User-Agent:", r.UserAgent())
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")
    query = strings.TrimSpace(query)

    totalStart := time.Now()
    fmt.Printf("[%s] Search started\n", query)

    // Empty query → empty state
    if query == "" {
        data := SearchPageData{Query: "", Cards: nil}
        templateName := "search.html"
        if r.Header.Get("HX-Request") == "true" {
            templateName = "search_results.html"
        }
        templates.ExecuteTemplate(w, templateName, data)
        return
    }

    apiStart := time.Now()
    words := strings.Fields(query)
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = "name:*" + w + "*"
	}
	queryString := strings.Join(parts, " ")

	queryEscaped := url.QueryEscape(queryString)
	apiURL := fmt.Sprintf("https://api.pokemontcg.io/v2/cards?q=%s&pageSize=10", queryEscaped)
    resp, err := http.Get(apiURL)
    if err != nil {
        serverError(w, err, "API call failed")
        return
    }
    defer resp.Body.Close()
    fmt.Printf("[%s] API call: %v\n", query, time.Since(apiStart))

    decodeStart := time.Now()
    var apiResp APIResponse
    err = json.NewDecoder(resp.Body).Decode(&apiResp)
    if err != nil {
        serverError(w, err, "JSON decode failed")
        return
    }
    fmt.Printf("[%s] JSON decode: %v (got %d cards)\n", query, time.Since(decodeStart), len(apiResp.Data))

    renderStart := time.Now()
    data := SearchPageData{
        Query: query,
        Cards: apiResp.Data,
    }

    templateName := "search.html"
    if r.Header.Get("HX-Request") == "true" {
        templateName = "search_results.html"
    }

    err = templates.ExecuteTemplate(w, templateName, data)
    if err != nil {
        serverError(w, err, "Template render failed")
    }
    fmt.Printf("[%s] Template render: %v\n", query, time.Since(renderStart))
    fmt.Printf("[%s] TOTAL: %v\n\n", query, time.Since(totalStart))
}

func cardHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	url := fmt.Sprintf("https://api.pokemontcg.io/v2/cards/%s", id)
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	var apiResp SingleCardResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templateName := "card.html"
	if r.Header.Get("HX-Request") == "true" {
		templateName = "card_detail.html"
	}

	err = templates.ExecuteTemplate(w, templateName, apiResp.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func serverError(w http.ResponseWriter, err error, msg string) {
    log.Printf("ERROR: %s: %v", msg, err)
    http.Error(w, "Something went wrong. Please try again.", http.StatusInternalServerError)
}
