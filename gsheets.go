package main

import (
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"log"
)

func newSpreadsheetService() *sheets.Service {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(gsheetsCredsPath), option.WithScopes(sheets.SpreadsheetsScope))

	if err != nil {
		log.Fatalf("Unable to retrieve Sheets Client %v", err)
	}

	return srv
}

//TODO(chriswatkins) getValuesFromSpreadsheet function to read contacts list

func writeToSpreadsheet(s *sheets.Service, request *SheetsPushRequest) error {

	var vr sheets.ValueRange
	vr.Values = request.Values

	log.Println("Writing the following ValueRange values", vr.Values)

	res, err := s.Spreadsheets.Values.Append(request.SpreadsheetID, request.Range, &vr).ValueInputOption("USER_ENTERED").Do()

	fmt.Println("spreadsheet push ", res)

	if err != nil {
		fmt.Println("Unable to update data to sheet  ", err)
	}

	return err
}

// SheetsRequestProcessor accepts incoming requests via the sheetChan channel and writes to the connected Sheet
func SheetsRequestProcessor(sheetsChan <-chan SheetsPushRequest) {
	sheetsService := newSpreadsheetService()

	for request := range sheetsChan {
		err := writeToSpreadsheet(sheetsService, &request)

		if err != nil {
			log.Fatal("Failed to write value to Sheet")
		}
	}
}

// SheetsPushRequest is a struct to record values and destinations of data to be written to a Google Sheet
type SheetsPushRequest struct {
	SpreadsheetID string          `json:"spreadsheet_id"`
	Range         string          `json:"range"`
	Values        [][]interface{} `json:"values"`
}
