package google

import (
	"fmt"
	"time"

	"google.golang.org/api/sheets/v4"
)

type SheetsClient struct {
	service     *sheets.Service
	spreadsheetID string
}

type TimeEntry struct {
	Date        string
	Project     string
	Task        string
	Hours       float64
	Description string
	GitCommits  string
	GitPRs      string
}

func NewSheetsClient(service *sheets.Service, spreadsheetID string) *SheetsClient {
	return &SheetsClient{
		service:     service,
		spreadsheetID: spreadsheetID,
	}
}

func (s *SheetsClient) AppendTimeEntry(entry TimeEntry) error {
	values := [][]interface{}{
		{
			entry.Date,
			entry.Project,
			entry.Task,
			entry.Hours,
			entry.Description,
			entry.GitCommits,
			entry.GitPRs,
		},
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := s.service.Spreadsheets.Values.Append(
		s.spreadsheetID,
		"A:G",
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("unable to append data to sheet: %v", err)
	}

	return nil
}

func (s *SheetsClient) GetTodayEntries() ([]TimeEntry, error) {
	today := time.Now().Format("2006-01-02")
	
	resp, err := s.service.Spreadsheets.Values.Get(
		s.spreadsheetID,
		"A:G",
	).Do()
	
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	var entries []TimeEntry
	for _, row := range resp.Values {
		if len(row) > 0 && row[0] == today {
			entry := TimeEntry{
				Date: getStringValue(row, 0),
				Project: getStringValue(row, 1),
				Task: getStringValue(row, 2),
			}
			
			if len(row) > 3 {
				if hours, ok := row[3].(float64); ok {
					entry.Hours = hours
				}
			}
			
			entry.Description = getStringValue(row, 4)
			entry.GitCommits = getStringValue(row, 5)
			entry.GitPRs = getStringValue(row, 6)
			
			entries = append(entries, entry)
		}
	}
	
	return entries, nil
}

func (s *SheetsClient) GetWeekEntries() ([]TimeEntry, error) {
	now := time.Now()
	weekStart := now.AddDate(0, 0, -int(now.Weekday()))
	weekEnd := weekStart.AddDate(0, 0, 7)
	
	resp, err := s.service.Spreadsheets.Values.Get(
		s.spreadsheetID,
		"A:G",
	).Do()
	
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	var entries []TimeEntry
	for _, row := range resp.Values {
		if len(row) > 0 {
			dateStr := getStringValue(row, 0)
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			
			if date.After(weekStart) && date.Before(weekEnd) {
				entry := TimeEntry{
					Date: dateStr,
					Project: getStringValue(row, 1),
					Task: getStringValue(row, 2),
				}
				
				if len(row) > 3 {
					if hours, ok := row[3].(float64); ok {
						entry.Hours = hours
					}
				}
				
				entry.Description = getStringValue(row, 4)
				entry.GitCommits = getStringValue(row, 5)
				entry.GitPRs = getStringValue(row, 6)
				
				entries = append(entries, entry)
			}
		}
	}
	
	return entries, nil
}

func getStringValue(row []interface{}, index int) string {
	if len(row) > index {
		if val, ok := row[index].(string); ok {
			return val
		}
	}
	return ""
}