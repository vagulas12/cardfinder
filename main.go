package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

var templates = template.Must(template.ParseGlob("templates/*.html"))

type Card struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIResponse struct {
	Data []Card `json:"data"`
}

type SearchPageData struct {
	Query string
	Cards []Card
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/about", aboutHandler)
	http.HandleFunc("/search", searchHandler)
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
	url := fmt.Sprintf("https://api.pokemontcg.io/v2/cards?q=name:%s&pageSize=10", query)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
