package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

var verifyToken = os.Getenv("VERIFY_TOKEN")
var pageAccessToken = os.Getenv("PAGE_ACCESS_TOKEN")
var listenPort = os.Getenv("PORT")
var sheetsID = os.Getenv("GSHEET_DB_ID")

// FacebookMessage defines the content of a Facebook message itself
type FacebookMessage struct {
	Sender struct {
		ID string `json:"id,omitempty"`
	} `json:"sender,omitempty"`

	Recipient struct {
		ID string `json:"id,omitempty"`
	} `json:"recipient,omitempty"`

	Timestamp     int    `json:"timestamp,omitempty"`
	MessagingType string `json:"messaging_type,omitempty"`

	Message struct {
		Text string `json:"text,omitempty"`
	} `json:"message,omitempty"`
}

// FacebookMessagingEvent contains the content of a messaging event
// This contains minimal metadata and then the core content of a FacebookMessage
type FacebookMessagingEvent struct {
	Object string `json:"object,omitempty"`
	Entry  []struct {
		ID        string            `json:"id,omitempty"`
		Time      int               `json:"time,omitempty"`
		Messaging []FacebookMessage `json:"messaging,omitempty"`
	} `json:"entry,omitempty"`
}

// IncomingMessageChan accepts incoming FacebookMessages for processing
var IncomingMessageChan = make(chan FacebookMessage)

// OutgoingMessageChan accepts outgoing FacebookMessages to be sent
var OutgoingMessageChan = make(chan FacebookMessage)

// SheetsChan accepts outgoing SheetsPushRequests to be written
var SheetsChan = make(chan SheetsPushRequest)

func main() {

	go IncomingMessageProcessor(IncomingMessageChan)
	log.Println("Started incoming message processor")

	go OutgoingMessageProcessor(OutgoingMessageChan)
	log.Println("Started outgoing message processor")

	go SheetsRequestProcessor(SheetsChan)
	log.Println("Started Sheets request processor")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/webhook", webhookHandler)

	log.Println("Starting HTTP server on port " + listenPort)
	log.Println(http.ListenAndServe(":"+listenPort, nil))
}

// Simple Hello World handler for root server calls
func indexHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("Hello, World!"))

	if err != nil {
		log.Fatal(err)
	}

}

// Facebook will send stuff to the the webhook endpoint
func webhookHandler(w http.ResponseWriter, r *http.Request) {

	// GET methods are authentication routines
	if r.Method == "GET" {
		err := r.ParseForm()

		if err != nil {
			log.Fatal(err)
		}

		mode := r.Form.Get("hub.mode")
		token := r.Form.Get("hub.verify_token")
		challenge := r.Form.Get("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			log.Println("Authenticated successfully. Sending challenge back")
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(challenge))

			if err != nil {
				log.Fatal(err)
			}

		} else {
			w.WriteHeader(http.StatusForbidden)
			_, err := w.Write([]byte("Wrong mode or token"))

			if err != nil {
				log.Fatal(err)
			}

		}

	}

	// POST messages are actual events to process, post-authentication calls
	if r.Method == "POST" {

		receivedMessage := FacebookMessagingEvent{}
		jsonErr := json.NewDecoder(r.Body).Decode(&receivedMessage)

		if jsonErr != nil {
			log.Print("JSON parsing error")
			log.Fatal(jsonErr)
		}

		log.Println("Parsed the following message")
		log.Printf("%+v\n", receivedMessage)

		// Respond to FB with receipt acknowledgement, process a response
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("Received okay."))

		if err != nil {
			log.Fatal(err)
		}

		log.Print("Acknowledged message receipt.")

		for _, entry := range receivedMessage.Entry {
			// Pass first messaging object directly as a webook event will only contain 1
			// https://developers.facebook.com/docs/messenger-platform/reference/webhook-events

			// Add the message to our queue for processing
			IncomingMessageChan <- entry.Messaging[0]
			log.Println("Added message to queue for processing.")
		}
	}
}
