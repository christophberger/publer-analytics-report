package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	API struct {
		BaseURL   string `yaml:"base_url"`
		APIKeyEnv string `yaml:"api_key_env"`
		Model     string `yaml:"model"`
	} `yaml:"api"`
}

type OverviewData struct {
	WorkspaceName  string
	Followers      int
	Reach          int
	ReachRate      float64
	Engagements    int
	EngagementRate float64
	TopCountries   []CountryData
}

type CountryData struct {
	Country    string
	Users      int
	Percentage float64
}

type PostData struct {
	Date             string
	SocialAccount    string
	SocialNetwork    string
	PostLink         string
	PostText         string
	PostType         string
	Reach            int
	ReachRate        float64
	Reactions        int
	Comments         int
	Shares           int
	EngagementRate   float64
	LinkClicks       int
	ClickThroughRate float64
}

type HashtagData struct {
	Hashtag    string
	Score      float64
	Reach      int
	Reactions  int
	Comments   int
	Shares     int
	VideoViews int
}

type ReportData struct {
	Month                string
	Period               string
	Followers            int
	FollowersChange      int
	Reach                int
	ReachChange          float64
	Engagements          int
	EngagementsChange    float64
	EngagementRate       float64
	EngagementRateChange float64
	TopPosts             []PostData
	TopHashtags          []HashtagData
	TopCountries         []CountryData
	Insights             string
	NextSteps            string
}

func findCSVFiles(param string) (string, string, string, error) {
	info, err := os.Stat(param)
	if err != nil {
		return "", "", "", err
	}

	if info.IsDir() {
		return findCSVFilesInDir(param)
	} else {
		return findCSVFilesFromFile(param)
	}
}

func findCSVFilesInDir(dir string) (string, string, string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", "", "", err
	}

	var overview, posts, hashtags string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		filepath := dir + string(os.PathSeparator) + filename

		if isOverviewFile(filename) {
			overview = filepath
		} else if isPostInsightsFile(filename) {
			posts = filepath
		} else if isHashtagAnalysisFile(filename) {
			hashtags = filepath
		}
	}

	if overview == "" || posts == "" || hashtags == "" {
		return "", "", "", fmt.Errorf("could not find all required CSV files in directory: %s", dir)
	}

	return overview, posts, hashtags, nil
}

func findCSVFilesFromFile(filepath string) (string, string, string, error) {
	dir := filepath[:strings.LastIndex(filepath, string(os.PathSeparator))]

	if dir == "" {
		dir = "."
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return "", "", "", err
	}

	var overview, posts, hashtags string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		currentFilename := file.Name()
		currentFilepath := dir + string(os.PathSeparator) + currentFilename

		if isOverviewFile(currentFilename) {
			overview = currentFilepath
		} else if isPostInsightsFile(currentFilename) {
			posts = currentFilepath
		} else if isHashtagAnalysisFile(currentFilename) {
			hashtags = currentFilepath
		}
	}

	if overview == "" || posts == "" || hashtags == "" {
		return "", "", "", fmt.Errorf("could not find all required CSV files in directory: %s", dir)
	}

	return overview, posts, hashtags, nil
}

func isOverviewFile(filename string) bool {
	return strings.Contains(filename, "Overview") && strings.HasSuffix(filename, ".csv")
}

func isPostInsightsFile(filename string) bool {
	return strings.Contains(filename, "Post Insights") && strings.HasSuffix(filename, ".csv")
}

func isHashtagAnalysisFile(filename string) bool {
	return strings.Contains(filename, "Hashtag Analysis") && strings.HasSuffix(filename, ".csv")
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <file-or-directory>")
	}

	param := os.Args[1]

	overviewFile, postsFile, hashtagFile, err := findCSVFiles(param)
	if err != nil {
		log.Fatalf("Error finding CSV files: %v", err)
	}

	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	overviewData, err := readOverviewFile(overviewFile)
	if err != nil {
		log.Fatalf("Error reading overview file: %v", err)
	}

	postsData, err := readPostInsightsFile(postsFile)
	if err != nil {
		log.Fatalf("Error reading post insights file: %v", err)
	}

	hashtagData, err := readHashtagAnalysisFile(hashtagFile)
	if err != nil {
		log.Fatalf("Error reading hashtag analysis file: %v", err)
	}

	reportData := prepareReportData(overviewData, postsData, hashtagData)

	insights, err := generateInsights(reportData, config)
	if err != nil {
		log.Printf("Warning: Could not generate insights: %v", err)
		insights = "Insights generation failed. Please check API configuration."
	}

	nextSteps, err := generateNextSteps(reportData, config)
	if err != nil {
		log.Printf("Warning: Could not generate next steps: %v", err)
		nextSteps = "Next steps generation failed. Please check API configuration."
	}

	reportData.Insights = insights
	reportData.NextSteps = nextSteps

	err = generateReport(reportData)
	if err != nil {
		log.Fatalf("Error generating report: %v", err)
	}

	fmt.Printf("Report generated successfully: %s %s.md\n", overviewData.WorkspaceName, "2025-07")
}

func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func readOverviewFile(filename string) (*OverviewData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Skip header lines (3 lines)
	for i := 0; i < 3; i++ {
		_, err = reader.Read()
		if err != nil {
			return nil, err
		}
	}

	// Read main data row
	record, err := reader.Read()
	if err != nil {
		return nil, err
	}

	data := &OverviewData{
		WorkspaceName: record[0],
	}

	// Parse numeric values
	if len(record) > 2 {
		fmt.Sscanf(strings.TrimSpace(record[2]), "%d", &data.Followers)
	}
	if len(record) > 3 {
		fmt.Sscanf(strings.TrimSpace(record[3]), "%d", &data.Reach)
	}
	if len(record) > 4 {
		fmt.Sscanf(strings.TrimSpace(record[4]), "%f", &data.ReachRate)
	}
	if len(record) > 6 {
		fmt.Sscanf(strings.TrimSpace(record[6]), "%d", &data.Engagements)
	}
	if len(record) > 7 {
		rateStr := strings.TrimSpace(strings.TrimSuffix(record[7], "%"))
		fmt.Sscanf(rateStr, "%f", &data.EngagementRate)
	}

	// Read empty line
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	// Read country headers
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	// Read country data
	for i := 0; i < 10; i++ { // Read up to 10 countries
		record, err := reader.Read()
		if err != nil || len(record) < 2 {
			break
		}
		if record[0] == "" || strings.HasPrefix(record[0], "Top") {
			break
		}

		country := CountryData{
			Country: strings.TrimSpace(record[0]),
		}
		if record[1] != "" {
			fmt.Sscanf(strings.TrimSpace(record[1]), "%d", &country.Users)
		}
		data.TopCountries = append(data.TopCountries, country)
	}

	return data, nil
}

func readPostInsightsFile(filename string) ([]PostData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Skip header lines (4 lines)
	for i := 0; i < 4; i++ {
		_, err = reader.Read()
		if err != nil {
			return nil, err
		}
	}

	var posts []PostData
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed records
		}

		if len(record) < 8 {
			continue
		}

		post := PostData{
			PostType: strings.TrimSpace(record[5]),
		}

		// Only include Status posts for top-performing posts
		if post.PostType == "Status" {
			post.PostText = strings.TrimSpace(record[4])
			if record[8] != "" && record[8] != "-" {
				fmt.Sscanf(strings.TrimSpace(record[8]), "%d", &post.Reactions)
			}
			posts = append(posts, post)
		}
	}

	return posts, nil
}

func readHashtagAnalysisFile(filename string) ([]HashtagData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Skip header lines (4 lines)
	for i := 0; i < 4; i++ {
		_, err = reader.Read()
		if err != nil {
			return nil, err
		}
	}

	var hashtags []HashtagData
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed records
		}

		if len(record) < 6 {
			continue
		}

		hashtag := HashtagData{
			Hashtag: strings.TrimSpace(record[0]),
		}
		if record[4] != "" {
			fmt.Sscanf(strings.TrimSpace(record[4]), "%f", &hashtag.Score)
		}
		if record[5] != "" {
			fmt.Sscanf(strings.TrimSpace(record[5]), "%d", &hashtag.Reach)
		}
		if record[6] != "" {
			fmt.Sscanf(strings.TrimSpace(record[6]), "%d", &hashtag.Reactions)
		}
		if record[7] != "" {
			fmt.Sscanf(strings.TrimSpace(record[7]), "%d", &hashtag.Comments)
		}
		if record[8] != "" {
			fmt.Sscanf(strings.TrimSpace(record[8]), "%d", &hashtag.Shares)
		}
		if record[9] != "" {
			fmt.Sscanf(strings.TrimSpace(record[9]), "%d", &hashtag.VideoViews)
		}

		hashtags = append(hashtags, hashtag)
	}

	return hashtags, nil
}

func prepareReportData(overview *OverviewData, posts []PostData, hashtags []HashtagData) *ReportData {
	data := &ReportData{
		Month:                "July 2025",
		Period:               "1 Jul 2025 - 31 Jul 2025",
		Followers:            overview.Followers,
		FollowersChange:      0, // TODO: Calculate from previous period
		Reach:                overview.Reach,
		ReachChange:          0.0, // TODO: Calculate from previous period
		Engagements:          overview.Engagements,
		EngagementsChange:    0.0, // TODO: Calculate from previous period
		EngagementRate:       overview.EngagementRate,
		EngagementRateChange: 0.0, // TODO: Calculate from previous period
		TopCountries:         overview.TopCountries,
	}

	// Sort posts by reactions (descending)
	for i := 0; i < len(posts)-1; i++ {
		for j := i + 1; j < len(posts); j++ {
			if posts[i].Reactions < posts[j].Reactions {
				posts[i], posts[j] = posts[j], posts[i]
			}
		}
	}

	// Take top 5 posts
	if len(posts) > 5 {
		data.TopPosts = posts[:5]
	} else {
		data.TopPosts = posts
	}

	// Sort hashtags by reach (descending)
	for i := 0; i < len(hashtags)-1; i++ {
		for j := i + 1; j < len(hashtags); j++ {
			if hashtags[i].Reach < hashtags[j].Reach {
				hashtags[i], hashtags[j] = hashtags[j], hashtags[i]
			}
		}
	}

	// Take top 5 hashtags
	if len(hashtags) > 5 {
		data.TopHashtags = hashtags[:5]
	} else {
		data.TopHashtags = hashtags
	}

	return data
}

func generateInsights(data *ReportData, config *Config) (string, error) {
	prompt := fmt.Sprintf(`Based on the following social media analytics data for July 2025:

- Followers: %d
- Reach: %d
- Engagements: %d
- Engagement Rate: %.2f%%
- Top performing posts: %d posts with high engagement
- Top hashtags: %d hashtags analyzed

Please provide insights and recommendations for improving social media performance. Focus on what's working well and what could be improved.`,
		data.Followers, data.Reach, data.Engagements, data.EngagementRate, len(data.TopPosts), len(data.TopHashtags))

	return callOpenAI(prompt, config)
}

func generateNextSteps(data *ReportData, config *Config) (string, error) {
	prompt := fmt.Sprintf(`Based on the social media analytics data for July 2025:

- Followers: %d
- Reach: %d  
- Engagements: %d
- Engagement Rate: %.2f%%

Please suggest specific next steps and action items to optimize KPIs for the next month. Include concrete, actionable recommendations.`,
		data.Followers, data.Reach, data.Engagements, data.EngagementRate)

	return callOpenAI(prompt, config)
}

func callOpenAI(prompt string, config *Config) (string, error) {
	apiKey := os.Getenv(config.API.APIKeyEnv)
	if apiKey == "" {
		return "", fmt.Errorf("API key environment variable %s not set", config.API.APIKeyEnv)
	}

	// Create proper JSON request body using struct and marshaling
	type Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type Request struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		MaxTokens   int       `json:"max_tokens"`
		Temperature float64   `json:"temperature"`
	}

	request := Request{
		Model: config.API.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   500,
		Temperature: 0.7,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	req, err := http.NewRequest("POST", config.API.BaseURL+"/chat/completions", strings.NewReader(string(requestBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Parse JSON response properly
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

func generateReport(data *ReportData) error {
	tmpl := `# {{.Month}} KPIs

For the period {{.Period}}

## Monthly Performance Summary

- Total Followers: {{.Followers}} (+/-{{.FollowersChange}} new/fewer followers)
- Total Reach: {{.Reach}} (+/-{{.ReachChange}}% increase/decrease)
- Total Engagements: {{.Engagements}} (+/-{{.EngagementsChange}}% increase/decrease)
- Engagement Rate: {{.EngagementRate}}% (+/-{{.EngagementRateChange}}% increase/decrease)

## Interaction Breakdown

### Top-Performing Posts by Reactions

{{range $i, $post := .TopPosts}}{{if ge $i 5}}{{break}}{{end}}{{if eq $post.PostType "Status"}}
{{add $i 1}}. {{truncate $post.PostText 50}} ({{$post.Reactions}})
{{end}}{{end}}

### Top Hashtags by Score

{{range $i, $hashtag := .TopHashtags}}{{if ge $i 5}}{{break}}{{end}}
{{add $i 1}}. {{$hashtag.Hashtag}} ({{$hashtag.Score}})
{{end}}

### Geographic Distribution

{{range $i, $country := .TopCountries}}{{if ge $i 5}}{{break}}{{end}}
{{add $i 1}}. {{$country.Country}} ({{$country.Users}})
{{end}}

## Insights and Recommendations

{{.Insights}}

## Next Steps

{{.NextSteps}}
`

	// Custom template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"truncate": func(s string, length int) string {
			if len(s) <= length {
				return s
			}
			return s[:length] + "..."
		},
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return err
	}

	filename := "Safe Swiss Cloud 2025-07.md"
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, data)
}
