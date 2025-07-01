package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	g "github.com/serpapi/google-search-results-golang"
)

// Structs for AI Overview
type SearchMetadata struct {
	PageToken   string `json:"page_token"`
	SerpapiLink string `json:"serpapi_link"`
}

type AIOverview struct {
	TextBlocks []TextBlock `json:"text_blocks"`
	References []Reference `json:"references"`
	Error      string      `json:"error"`
}

func (a AIOverview) IsEmpty() bool {
	return len(a.TextBlocks) == 0 && len(a.References) == 0
}

type TextBlock struct {
	Type                    string     `json:"type"`
	Snippet                 string     `json:"snippet,omitempty"`
	SnippetHighlightedWords []string   `json:"snippet_highlighted_words,omitempty"`
	ReferenceIndexes        []int      `json:"reference_indexes,omitempty"`
	List                    []ListItem `json:"list,omitempty"`
}

type ListItem struct {
	Title            string `json:"title"`
	Snippet          string `json:"snippet"`
	ReferenceIndexes []int  `json:"reference_indexes"`
}

type Reference struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
	Index   int    `json:"index"`
}

// HTML Template
var tmpl = `
<!DOCTYPE html>
<html>
<head>
	<title>AI Overview Search</title>
	<style>
		body { font-family: sans-serif; margin: 2rem auto; max-width: 800px; }
		textarea { width: 100%; }
		.text-block { margin-bottom: 1rem; padding: 1rem; background: #f9f9f9; border-radius: 8px; }
	</style>
</head>
<body>
	<h1>🔍 Google AI Overview via SerpAPI</h1>
	<form method="GET">
		<input type="text" name="q" placeholder="Enter a search keyword..." style="width:80%;" value="{{.Query}}" required />
		<button type="submit">Search</button>
	</form>
	{{if .AI}}
		<h2>🧠 AI Overview Result</h2>
		{{range .AI.TextBlocks}}
			<div class="text-block">
				<strong>{{.Type | title}}</strong>
				<p>{{.Snippet}}</p>
				{{if .List}}
					<ul>
					{{range .List}}
						<li><strong>{{.Title}}</strong> — {{.Snippet}}</li>
					{{end}}
					</ul>
				{{end}}
			</div>
		{{end}}
		<h2>🧠 References</h2>
		{{range .AI.References}}
			<div class="text-block">
			<strong>title: <a href="{{.Link}}">{{.Title}}</a></strong>
			<p>Snippet: {{.Snippet}}</p>
			<p>Source: {{.Source}}</p>
			<p>Index: {{.Index}}</p>
			</div>
		{{end}}
	{{else if .Query}}
		<p><em>No AI Overview found for: {{.Query}}</em></p>
	{{end}}
</body>
</html>
`

// Template func map
var funcMap = template.FuncMap{
	"title": strings.Title,
}

func main() {
	tpl := template.Must(template.New("index").Funcs(funcMap).Parse(tmpl))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		data := struct {
			Query string
			AI    *AIOverview
		}{Query: query}

		if query != "" {
			ai, err := fetchAIOverview(query)
			if err != nil {
				log.Println("❌", err)
				data.AI.Error = err.Error()
			} else {
				data.AI = ai
			}
		}

		err := tpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}
	})

	log.Println("🚀 Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func fetchAIOverview(query string) (*AIOverview, error) {
	apiKey := os.Getenv("api_key") // 🛑 Replace with your key

	// Step 1: Try with regular Google search engine
	param := map[string]string{
		"engine":        "google",
		"q":             query,
		"location":      "Indonesia",
		"google_domain": "google.com",
		"gl":            "id",
		"hl":            "id",
	}

	fmt.Printf("params query: %+v\n", param)
	fmt.Printf("print datenow 1: %+v\n", time.Now())
	search := g.NewGoogleSearch(param, apiKey)
	fmt.Printf("print datenow 2: %+v\n", time.Now())
	results, err := search.GetJSON()
	if err != nil {
		fmt.Printf("print datenow 3: %+v\n", time.Now())
		fmt.Printf("error when get json search %+v", err)
		return nil, err
	}

	fmt.Printf("print datenow 4: %+v\n", time.Now())

	// Step 2: Try direct AI Overview
	aiOverviewRaw, ok := results["ai_overview"]
	if !ok {
		fmt.Printf("print datenow 5: %+v\n", time.Now())
		log.Print("❌ AI Overview not found for this query")
		return nil, errors.New("ai overview not found")
	}

	fmt.Printf("print datenow 6: %+v %+v\n", time.Now(), aiOverviewRaw)

	jsonBytes, _ := json.Marshal(aiOverviewRaw)
	fmt.Printf("print datenow 7: %+v %+v\n", time.Now(), aiOverviewRaw)

	var overview AIOverview
	err = json.Unmarshal(jsonBytes, &overview)
	fmt.Printf("print datenow 8: %+v %+v\n", time.Now(), aiOverviewRaw)
	if err == nil && !overview.IsEmpty() {
		fmt.Printf("print datenow 9: %+v %+v %+v\n", time.Now(), aiOverviewRaw, overview)
		return &overview, nil
	}

	// fallback to use page_token
	var meta SearchMetadata
	fmt.Printf("print datenow 9: %+v %+v\n", time.Now(), aiOverviewRaw)
	if err := json.Unmarshal(jsonBytes, &meta); err != nil {
		fmt.Printf("print datenow 10: %+v %+v\n", time.Now(), aiOverviewRaw)
		return nil, err
	}

	fmt.Println("✅ page_token:", meta.PageToken)
	fmt.Println("🔗 serpapi_link:", meta.SerpapiLink)

	search = g.NewGoogleSearch(map[string]string{
		"engine":     "google_ai_overview",
		"page_token": meta.PageToken,
		"hl":         "id",
		"gl":         "id",
	}, apiKey)

	results, err = search.GetJSON()
	if err != nil {
		fmt.Println("Failed to fetch AI Overview detail:", err)
		return nil, err
	}

	aiOverviewRaw = results["ai_overview"]
	jsonBytes, _ = json.Marshal(aiOverviewRaw)

	var result AIOverview
	err = json.Unmarshal(jsonBytes, &result)
	if err != nil {
		fmt.Println("failed unmarshal second hit:", err)
		return nil, err
	}
	overview = result
	return &overview, nil
}
