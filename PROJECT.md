# Project Specification

## Goal

Deliverable: A CLI tool that produces an analytics report from CSV files.

## Input

Three CSV files containing analytics data from Publer, a social media management app:

- Overview file
- Post insights file
- Hashtag analysis file

Find three example CSV files in the `testdata/` directory:

- ACME Inc (Workspace) ∙ Hashtag Analysis ∙ 1 Jul 2025 - 31 Jul 2025.csv 
- ACME Inc (Workspace) ∙ Overview ∙ 1 Jul 2025 - 31 Jul 2025.csv
- ACME Inc (Workspace) ∙ Post Insights ∙ 1 Jul 2025 - 31 Jul 2025.csv

Note that they don't follow the correct CSV specification: - They have two or three lines above the table that don't belong to the tabular data.
- They might contain more than one table, separated by two empty lines. Every table has a header row.

NOTE: The tool you work on shall NOT process the files in `testdata/`. These are only meant as examples. The tool shall receive a single parameter on the command line.

- If the command line parameter is a file name or a path to a file name, the tool shall ensure the file is one of the CSV files - overview, hashtag analysis, or post insights - and find the other two files in the same directory.
- If the parameter is a path, the tool shall expect to find the three CSV files at this path. 

## Output

A Markdown file containing a report. Find an example in `testdata/report.md` with placeholders where the tool shall insert real data. (Use this to create a Go template file.)

## Processing

The tool shall read the three CSV files and save the tables inside the CSVs into Go structs. 

Also save the tables to SQLite tables. Use an extra column "period" for properly grouping the rows by month. Use "YYYY-MM" in this column to specify the period. Some of the data will be needed to calculate changes from the previous month.

Use `modernc.org/sqlite` as the database driver, as this one does not need CGO. 

Extract the necessary data from the structs to fill the placeholders in the report. For the fields that show the changes from the previous month, query the SQLite database for the previous month's data and calculate the change.

It shall write the report to a file whose name includes the Publer workspace name and the period in the form YYYY-MM. 

Example: ACME Inc 2025-07.md

This is just an example, the real file name should contain the name of the actual workspace and the actual year and month as found in the input file names. 

For the top-performing posts, the tool shall remove all newlines from the post text.

For the report sections

  "## Insights and Recommendations"

and

  "## Next Steps"

the tool shall call an OpenAI-API-compatible `/chat/completions` endpoint to fill the sections according to the requests in the placeholders. It shall use the `net/http` package for this. The tool shall use a yaml config file named `config.yaml` for setting up an enpoint to call: The base URL, the env var name containing the API key, and the model name.

