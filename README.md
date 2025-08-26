# Publer Analytics Report

A small CLI that reads Publer Analytics CSV files for a month and generates a concise Markdown report with KPIs, top posts/hashtags/countries, plus AI‑generated insights and next steps.

## Install

- Prebuilt binaries: Download the latest release for your OS/arch from the GitHub Releases page and place the binary on your PATH
- Go (requires Go >= 1.21):
  - `go install github.com/christophberger/publer-analytics-report@latest`
- From source:
  - `git clone https://github.com/christophberger/publer-analytics-report`
  - `cd publer-analytics-report`
  - `go build -o publer-analytics-report ./...`

## Configure

Create or edit `config.yaml` in the working directory:

```yaml
api:
  base_url: "https://api.openai.com/v1"   # OpenAI-compatible endpoint
  api_key_env: "OPENAI_API_KEY"           # Name of the env var that holds the API key
  model: "gpt-3.5-turbo"                  # Model to use
```

- Set the environment variable referenced by `api_key_env`, for example:
  - macOS/Linux: `export OPENAI_API_KEY=...`
  - Windows (PowerShell): `$Env:OPENAI_API_KEY = "..."`
- To use a different provider, set `base_url` and `model` accordingly.

## Run

1) In Publer, manually download the three analytics CSVs for the previous month into a separate directory:
   - Overview
   - Post Insights
   - Hashtag Analysis

2) Execute the tool, passing the path to the directory that contains the CSV files:

```bash
publer-analytics-report /path/to/month-folder
```

3) The tool writes a Markdown file named like: `ACME Inc 2025-07.md` in the current directory.

4) Open your Google Doc and use "Paste from Markdown" to paste the generated report.

## Technical overview

- Parse CSVs: Read and analyze the tables inside the three Publer CSVs (Overview, Post Insights, Hashtag Analysis)
- Persist data: Store each month's data in a local SQLite database file `analytics.db`
- Compute stats: Generate monthly KPIs and a few month-over-month comparisons from the stored data
- LLM insights: Call a configured LLM to generate human‑readable insights and recommended next steps

Notes
- If the LLM call fails or no API key is set, the report still generates with placeholder text in the Insights/Next Steps sections
- The output is plain Markdown designed for easy pasting into Google Docs
