package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	_ "modernc.org/sqlite"
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

func extractDateFromFilename(filename string) (string, error) {
	parts := strings.Split(filename, "∙")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid filename format")
	}

	datePart := strings.TrimSpace(parts[len(parts)-1])
	datePart = strings.TrimSuffix(datePart, ".csv")

	dateRange := strings.Split(datePart, "-")
	if len(dateRange) < 1 {
		return "", fmt.Errorf("invalid date format in filename")
	}

	startDate := strings.TrimSpace(dateRange[0])
	dateComponents := strings.Fields(startDate)
	if len(dateComponents) < 3 {
		return "", fmt.Errorf("invalid start date format")
	}

	month := dateComponents[1]
	year := dateComponents[2]

	monthMap := map[string]string{
		"Jan": "01", "Feb": "02", "Mar": "03", "Apr": "04", "May": "05", "Jun": "06",
		"Jul": "07", "Aug": "08", "Sep": "09", "Oct": "10", "Nov": "11", "Dec": "12",
	}

	monthNum, ok := monthMap[month]
	if !ok {
		return "", fmt.Errorf("invalid month: %s", month)
	}

	return fmt.Sprintf("%s-%s", year, monthNum), nil
}

func extractPeriodFromFilename(filename string) string {
	parts := strings.Split(filename, "∙")
	if len(parts) < 3 {
		return "Unknown Period"
	}

	datePart := strings.TrimSpace(parts[len(parts)-1])
	datePart = strings.TrimSuffix(datePart, ".csv")

	return datePart
}

func extractMonthFromFilename(filename string) string {
	period := extractPeriodFromFilename(filename)

	dateRange := strings.Split(period, "-")
	if len(dateRange) < 1 {
		return "Unknown Month"
	}

	startDate := strings.TrimSpace(dateRange[0])
	dateComponents := strings.Fields(startDate)
	if len(dateComponents) < 3 {
		return "Unknown Month"
	}

	month := dateComponents[1]
	year := dateComponents[2]

	monthNames := map[string]string{
		"Jan": "January", "Feb": "February", "Mar": "March", "Apr": "April", "May": "May", "Jun": "June",
		"Jul": "July", "Aug": "August", "Sep": "September", "Oct": "October", "Nov": "November", "Dec": "December",
	}

	monthName, ok := monthNames[month]
	if !ok {
		return "Unknown Month"
	}

	return fmt.Sprintf("%s %s", monthName, year)
}

func generateReportFilename(workspaceName, overviewFile string) (string, error) {
	datePart, err := extractDateFromFilename(overviewFile)
	if err != nil {
		return "", err
	}

	cleanWorkspace := strings.ReplaceAll(workspaceName, "(Workspace)", "")
	cleanWorkspace = strings.TrimSpace(cleanWorkspace)

	return fmt.Sprintf("%s %s.md", cleanWorkspace, datePart), nil
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
		fp := filepath.Join(dir, filename)

		if isOverviewFile(filename) {
			overview = fp
		} else if isPostInsightsFile(filename) {
			posts = fp
		} else if isHashtagAnalysisFile(filename) {
			hashtags = fp
		}
	}

	if overview == "" || posts == "" || hashtags == "" {
		return "", "", "", fmt.Errorf("could not find all required CSV files in directory: %s", dir)
	}

	return overview, posts, hashtags, nil
}

func findCSVFilesFromFile(filePath string) (string, string, string, error) {
	dir := filepath.Dir(filePath)
	if dir == "" {
		dir = "."
	}

	base := filepath.Base(filePath)
	if !(isOverviewFile(base) || isPostInsightsFile(base) || isHashtagAnalysisFile(base)) {
		return "", "", "", fmt.Errorf("provided file is not a recognized CSV type: %s", base)
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
		currentFilepath := filepath.Join(dir, currentFilename)

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

func openDB(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}

func initSchema(db *sql.DB) error {
	stmts := []string{
		"CREATE TABLE IF NOT EXISTS overview (workspace TEXT NOT NULL, period TEXT NOT NULL, followers INTEGER, reach INTEGER, reach_rate REAL, engagements INTEGER, engagement_rate REAL, PRIMARY KEY(workspace, period));",
		"CREATE TABLE IF NOT EXISTS countries (workspace TEXT NOT NULL, period TEXT NOT NULL, country TEXT NOT NULL, users INTEGER, percentage REAL);",
		"CREATE TABLE IF NOT EXISTS posts (workspace TEXT NOT NULL, period TEXT NOT NULL, post_text TEXT, post_type TEXT, reactions INTEGER);",
		"CREATE TABLE IF NOT EXISTS hashtags (workspace TEXT NOT NULL, period TEXT NOT NULL, hashtag TEXT NOT NULL, score REAL, reach INTEGER, reactions INTEGER, comments INTEGER, shares INTEGER, video_views INTEGER);",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

func saveOverview(db *sql.DB, period string, data *OverviewData) error {
	_, err := db.Exec(
		"INSERT INTO overview(workspace, period, followers, reach, reach_rate, engagements, engagement_rate) VALUES(?,?,?,?,?,?,?) ON CONFLICT(workspace, period) DO UPDATE SET followers=excluded.followers, reach=excluded.reach, reach_rate=excluded.reach_rate, engagements=excluded.engagements, engagement_rate=excluded.engagement_rate",
		data.WorkspaceName, period, data.Followers, data.Reach, data.ReachRate, data.Engagements, data.EngagementRate,
	)
	return err
}

func saveCountries(db *sql.DB, period string, workspace string, countries []CountryData) error {
	if _, err := db.Exec("DELETE FROM countries WHERE workspace=? AND period=?", workspace, period); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO countries(workspace, period, country, users, percentage) VALUES(?,?,?,?,?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, c := range countries {
		if _, err := stmt.Exec(workspace, period, c.Country, c.Users, c.Percentage); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	stmt.Close()
	return tx.Commit()
}

func savePosts(db *sql.DB, period string, workspace string, posts []PostData) error {
	if _, err := db.Exec("DELETE FROM posts WHERE workspace=? AND period=?", workspace, period); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO posts(workspace, period, post_text, post_type, reactions) VALUES(?,?,?,?,?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, p := range posts {
		if _, err := stmt.Exec(workspace, period, p.PostText, p.PostType, p.Reactions); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	stmt.Close()
	return tx.Commit()
}

func saveHashtags(db *sql.DB, period string, workspace string, hashtags []HashtagData) error {
	if _, err := db.Exec("DELETE FROM hashtags WHERE workspace=? AND period=?", workspace, period); err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("INSERT INTO hashtags(workspace, period, hashtag, score, reach, reactions, comments, shares, video_views) VALUES(?,?,?,?,?,?,?,?,?)")
	if err != nil {
		tx.Rollback()
		return err
	}
	for _, h := range hashtags {
		if _, err := stmt.Exec(workspace, period, h.Hashtag, h.Score, h.Reach, h.Reactions, h.Comments, h.Shares, h.VideoViews); err != nil {
			stmt.Close()
			tx.Rollback()
			return err
		}
	}
	stmt.Close()
	return tx.Commit()
}

func previousPeriod(period string) (string, error) {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return "", err
	}
	prev := t.AddDate(0, -1, 0)
	return prev.Format("2006-01"), nil
}

func getPreviousOverview(db *sql.DB, workspace, period string) (*OverviewData, error) {
	row := db.QueryRow("SELECT followers, reach, reach_rate, engagements, engagement_rate FROM overview WHERE workspace=? AND period=?", workspace, period)
	var followers, reach, engagements int
	var reachRate, engagementRate float64
	err := row.Scan(&followers, &reach, &reachRate, &engagements, &engagementRate)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &OverviewData{WorkspaceName: workspace, Followers: followers, Reach: reach, ReachRate: reachRate, Engagements: engagements, EngagementRate: engagementRate}, nil
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

	period, err := extractDateFromFilename(overviewFile)
	if err != nil {
		log.Fatalf("Error extracting period from filename: %v", err)
	}

	db, err := openDB("analytics.db")
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	if err := initSchema(db); err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	if err := saveOverview(db, period, overviewData); err != nil {
		log.Fatalf("Error saving overview: %v", err)
	}
	if err := saveCountries(db, period, overviewData.WorkspaceName, overviewData.TopCountries); err != nil {
		log.Fatalf("Error saving countries: %v", err)
	}
	if err := savePosts(db, period, overviewData.WorkspaceName, postsData); err != nil {
		log.Fatalf("Error saving posts: %v", err)
	}
	if err := saveHashtags(db, period, overviewData.WorkspaceName, hashtagData); err != nil {
		log.Fatalf("Error saving hashtags: %v", err)
	}

	reportData := prepareReportData(db, overviewData, postsData, hashtagData, overviewFile)

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

	reportFilename, err := generateReportFilename(overviewData.WorkspaceName, overviewFile)
	if err != nil {
		log.Fatalf("Error generating report filename: %v", err)
	}

	err = generateReport(reportData, reportFilename)
	if err != nil {
		log.Fatalf("Error generating report: %v", err)
	}

	fmt.Printf("Report generated successfully: %s\n", reportFilename)
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
	reader.FieldsPerRecord = -1

	var rec []string
	for {
		rec, err = reader.Read()
		if err != nil {
			return nil, err
		}
		if len(rec) > 0 && strings.HasPrefix(strings.TrimSpace(rec[0]), "Workspace Name") {
			break
		}
	}

	rec, err = reader.Read()
	if err != nil {
		return nil, err
	}

	data := &OverviewData{WorkspaceName: strings.TrimSpace(rec[0])}
	if len(rec) > 2 {
		fmt.Sscanf(strings.TrimSpace(rec[2]), "%d", &data.Followers)
	}
	if len(rec) > 3 {
		fmt.Sscanf(strings.TrimSpace(rec[3]), "%d", &data.Reach)
	}
	if len(rec) > 4 {
		fmt.Sscanf(strings.TrimSpace(rec[4]), "%f", &data.ReachRate)
	}
	if len(rec) > 6 {
		fmt.Sscanf(strings.TrimSpace(rec[6]), "%d", &data.Engagements)
	}
	if len(rec) > 7 {
		rateStr := strings.TrimSpace(strings.TrimSuffix(rec[7], "%"))
		fmt.Sscanf(rateStr, "%f", &data.EngagementRate)
	}

	for {
		rec, err = reader.Read()
		if err != nil {
			return data, nil
		}
		if len(rec) > 0 && strings.HasPrefix(strings.TrimSpace(rec[0]), "Top Countries") {
			break
		}
	}

	total := 0
	for {
		rec, err = reader.Read()
		if err != nil || len(rec) < 2 {
			break
		}
		name := strings.TrimSpace(rec[0])
		if name == "" || strings.HasPrefix(name, "Top") {
			break
		}
		country := CountryData{Country: name}
		if strings.TrimSpace(rec[1]) == "" {
			break
		}
		u, errNum := strconv.Atoi(strings.TrimSpace(rec[1]))
		if errNum != nil {
			break
		}
		country.Users = u
		total += country.Users
		data.TopCountries = append(data.TopCountries, country)
	}

	if total > 0 {
		for i := range data.TopCountries {
			data.TopCountries[i].Percentage = float64(data.TopCountries[i].Users) * 100.0 / float64(total)
		}
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
	reader.FieldsPerRecord = -1

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
			continue
		}

		if len(record) < 8 {
			continue
		}

		post := PostData{
			PostType: strings.TrimSpace(record[5]),
		}

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
	reader.FieldsPerRecord = -1

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
			continue
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

func prepareReportData(db *sql.DB, overview *OverviewData, posts []PostData, hashtags []HashtagData, overviewFile string) *ReportData {
	period := extractPeriodFromFilename(overviewFile)
	month := extractMonthFromFilename(overviewFile)

	data := &ReportData{
		Month:          month,
		Period:         period,
		Followers:      overview.Followers,
		Reach:          overview.Reach,
		Engagements:    overview.Engagements,
		EngagementRate: overview.EngagementRate,
		TopCountries:   overview.TopCountries,
	}

	sort.Slice(data.TopCountries, func(i, j int) bool { return data.TopCountries[i].Users > data.TopCountries[j].Users })
	if len(data.TopCountries) > 5 {
		data.TopCountries = data.TopCountries[:5]
	}

	sort.Slice(posts, func(i, j int) bool { return posts[i].Reactions > posts[j].Reactions })
	if len(posts) > 5 {
		data.TopPosts = posts[:5]
	} else {
		data.TopPosts = posts
	}

	sort.Slice(hashtags, func(i, j int) bool { return hashtags[i].Score > hashtags[j].Score })
	if len(hashtags) > 5 {
		data.TopHashtags = hashtags[:5]
	} else {
		data.TopHashtags = hashtags
	}

	currPeriod, err := extractDateFromFilename(overviewFile)
	if err == nil {
		if prevPeriod, perr := previousPeriod(currPeriod); perr == nil {
			if prev, qerr := getPreviousOverview(db, overview.WorkspaceName, prevPeriod); qerr == nil && prev != nil {
				data.FollowersChange = overview.Followers - prev.Followers
				if prev.Reach > 0 {
					data.ReachChange = float64(overview.Reach-prev.Reach) * 100.0 / float64(prev.Reach)
				}
				if prev.Engagements > 0 {
					data.EngagementsChange = float64(overview.Engagements-prev.Engagements) * 100.0 / float64(prev.Engagements)
				}
				data.EngagementRateChange = overview.EngagementRate - prev.EngagementRate
			}
		}
	}

	return data
}

func generateInsights(data *ReportData, config *Config) (string, error) {
	prompt := fmt.Sprintf(`Based on the following social media analytics data for %s (%s):

- Followers: %d
- Reach: %d
- Engagements: %d
- Engagement Rate: %.2f%%
- Top performing posts: %d posts with high engagement
- Top hashtags: %d hashtags analyzed

Please provide insights and recommendations for improving social media performance. Focus on what's working well and what could be improved.`,
		data.Month, data.Period, data.Followers, data.Reach, data.Engagements, data.EngagementRate, len(data.TopPosts), len(data.TopHashtags))

	return callOpenAI(prompt, config)
}

func generateNextSteps(data *ReportData, config *Config) (string, error) {
	prompt := fmt.Sprintf(`Based on the social media analytics data for %s (%s):

- Followers: %d
- Reach: %d  
- Engagements: %d
- Engagement Rate: %.2f%%

Please suggest specific next steps and action items to optimize KPIs for the next month. Include concrete, actionable recommendations.`,
		data.Month, data.Period, data.Followers, data.Reach, data.Engagements, data.EngagementRate)

	return callOpenAI(prompt, config)
}

func callOpenAI(prompt string, config *Config) (string, error) {
	apiKey := os.Getenv(config.API.APIKeyEnv)
	if apiKey == "" {
		return "", fmt.Errorf("API key environment variable %s not set", config.API.APIKeyEnv)
	}

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

	req, err := http.NewRequest("POST", config.API.BaseURL+"/chat/completions", bytes.NewReader(requestBody))
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

func generateReport(data *ReportData, filename string) error {
	tmpl := `# {{.Month}} KPIs

For the period {{.Period}}

## Monthly Performance Summary

- Total Followers: {{.Followers}} (+/-{{.FollowersChange}} new/fewer followers)
- Total Reach: {{.Reach}} (+/-{{printf "%.1f" .ReachChange}}% increase/decrease)
- Total Engagements: {{.Engagements}} (+/-{{printf "%.1f" .EngagementsChange}}% increase/decrease)
- Engagement Rate: {{.EngagementRate}}% (+/-{{printf "%.1f" .EngagementRateChange}}% increase/decrease)

## Interaction Breakdown

### Top-Performing Posts by Reactions

{{range $i, $post := .TopPosts}}
{{add $i 1}}. {{truncate $post.PostText 50}} ({{$post.Reactions}})
{{end}}

### Top Hashtags by Score

{{range $i, $hashtag := .TopHashtags}}
{{add $i 1}}. {{$hashtag.Hashtag}} ({{$hashtag.Score}})
{{end}}

### Geographic Distribution

{{range $i, $country := .TopCountries}}
{{add $i 1}}. {{$country.Country}} ({{printf "%.1f" $country.Percentage}}%)
{{end}}

## Insights and Recommendations

{{.Insights}}

## Next Steps

{{.NextSteps}}
`

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"truncate": func(s string, length int) string {
			clean := strings.ReplaceAll(s, "\n", " ")
			clean = strings.ReplaceAll(clean, "\r", " ")
			words := strings.Fields(clean)
			clean = strings.Join(words, " ")
			if len(clean) <= length { return clean }
			return clean[:length] + "..."
		},
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, data)
}
