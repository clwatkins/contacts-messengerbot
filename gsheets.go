package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
	"log"
	"os"
)

type googleCreds struct {
	Type                string `json:"type,omitempty"`
	ProjectID           string `json:"project_id,omitempty"`
	PrivateKeyID        string `json:"private_key_id,omitempty"`
	PrivateKey          string `json:"private_key,omitempty"`
	ClientEmail         string `json:"client_email,omitempty"`
	ClientID            string `json:"client_id,omitempty"`
	AuthURI             string `json:"auth_uri,omitempty"`
	TokenURI            string `json:"token_uri,omitempty"`
	AuthProviderCertURL string `json:"auth_provider_x509_cert_url,omitempty"`
	ClientCertURL       string `json:"client_x509_cert_url,omitempty"`
}

func newSpreadsheetService() *sheets.Service {
	ctx := context.Background()

	gcreds := googleCreds{
		os.Getenv("GCREDS_TYPE"),
		os.Getenv("GCREDS_PROJECT_ID"),
		os.Getenv("GCREDS_PRIVATE_KEY_ID"),
		os.Getenv("GCREDS_PRIVATE_KEY"),
		os.Getenv("GCREDS_CLIENT_EMAIL"),
		os.Getenv("GCREDS_CLIENT_ID"),
		os.Getenv("GCREDS_AUTH_URI"),
		os.Getenv("GCREDS_TOKEN_URI"),
		os.Getenv("GCREDS_AUTH_PROVIDER_CERT_URL"),
		os.Getenv("GCREDS_CLIENT_CERT_URL"),
	}

	gcredsJSON, err := json.Marshal(gcreds)

	if err != nil {
		log.Fatal(err)
	}

	srv, err := sheets.NewService(ctx, option.WithCredentialsJSON(gcredsJSON), option.WithScopes(sheets.SpreadsheetsScope))

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
