package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	StateClear                   = "CLEAR"
	StateAwaitingChoices         = "AWAITING_CHOICES"
	StateAwaitingConfirmationAdd = "AWAITING_CONFIRM_ADD"
	StateError                   = "ERROR"

	ActionSaw = "SAW"
	ActionAdd = "MET"
)

//TODO(chriswatkins) cache contacts list periodically
//TODO(chriswatkins) understand if someone is in the database

func processStateClear(message FacebookMessage) string {
	var peopleAndComment = strings.SplitN(message.Message.Text, "/", 2)
	var peopleSplit = strings.Split(peopleAndComment[0], ",")

	for _, person := range peopleSplit {
		sheetsRequest := SheetsPushRequest{SpreadsheetID: sheetsID, Range: "Database!A:D"}
		sheetsRequest.Values = [][]interface{}{{
			strings.Trim(person, " "),
			time.Now(),
			ActionSaw,
			strings.Trim(peopleAndComment[1], " "),
		}}

		SheetsChan <- sheetsRequest

	}

	responseMsg := FacebookMessage{MessagingType: "RESPONSE"}
	responseMsg.Message.Text = fmt.Sprintf("You just saw %d people!", len(peopleSplit))
	responseMsg.Recipient.ID = message.Sender.ID

	OutgoingMessageChan <- responseMsg

	return StateClear

}

// IncomingMessageProcessor accepts incoming FacebokMessage objects via its channel and generates appropriate responses
func IncomingMessageProcessor(incomingMessageChan <-chan FacebookMessage) {

	var processorState = StateClear

	// Will keep the go routine live as long as the channel is open
	for message := range incomingMessageChan {

		if processorState == StateClear {
			processorState = processStateClear(message)
		}

	}
}

func OutgoingMessageProcessor(outgoingMessageChan <-chan FacebookMessage) {

	for outgoingMessage := range outgoingMessageChan {
		responseURL, err := url.Parse("https://graph.facebook.com/v5.0/me/messages")

		if err != nil {
			log.Panic(err)
		}

		// Add access token as query param
		queryParams := url.Values{}
		queryParams.Add("access_token", pageAccessToken)
		responseURL.RawQuery = queryParams.Encode()

		jsonOutgoingMessage, _ := json.Marshal(outgoingMessage)

		log.Println("Returning following JSON")
		log.Printf("%+v\n", outgoingMessage)

		resp, err := http.Post(
			responseURL.String(),
			"application/json",
			bytes.NewBuffer(jsonOutgoingMessage),
		)

		if err != nil {
			log.Panic("Got an unexpected response on sending FB message", err)
		}

		log.Println("Received following acknowledgement from FB messenger")
		log.Print(resp)
	}
}
