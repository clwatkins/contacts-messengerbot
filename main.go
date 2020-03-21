package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
)

var verifyToken string = os.Getenv("VERIFY_TOKEN")
var pageAccessToken string = os.Getenv("PAGE_ACCESS_TOKEN")
var listenPort string = os.Getenv("PORT")

type facebookMessage struct {
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

type facebookMessagingEvent struct {
	Object string `json:"object,omitempty"`
	Entry  []struct {
		ID        string            `json:"id,omitempty"`
		Time      int               `json:"time,omitempty"`
		Messaging []facebookMessage `json:"messaging,omitempty"`
	} `json:"entry,omitempty"`
}

// StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Code    int
	Message string
}

// Open a chan that will act as a message queue
// Incoming messages will be added to the channel, with a worker routine for continuous processing
var messageChan = make(chan facebookMessage)

func main() {

	go messageProcessor(messageChan)
	log.Println("Started message processor")

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/webhook", webhookHandler)

	// Start the ListenAndService on a go routine to avoid blocking messageProcessor
	go func() {
		log.Println(http.ListenAndServe(":"+listenPort, nil))
	}()
	log.Println("Started HTTP server.")
}

// Simple Hello World handler for root server calls
func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hello, World!"))
}

// Facebook will send stuff to the the webook endpoint
func webhookHandler(w http.ResponseWriter, r *http.Request) {

	// GET methods are authentication routines
	if r.Method == "GET" {
		r.ParseForm()
		mode := r.Form.Get("hub.mode")
		token := r.Form.Get("hub.verify_token")
		challenge := r.Form.Get("hub.challenge")

		if mode == "subscribe" && token == verifyToken {
			log.Println("Authenticated successfully. Sending challenge back")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(challenge))
		} else {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Wrong mode or token"))
		}
	}

	// POST messages are actual events to process, post-authentication calls
	if r.Method == "POST" {

		// Save a copy of this request for debugging.
		// requestDump, err := httputil.DumpRequest(r, true)
		// if err != nil {
		// 	fmt.Println(err)
		// }
		// log.Print(string(requestDump))

		receivedMessage := facebookMessagingEvent{}
		jsonErr := json.NewDecoder(r.Body).Decode(&receivedMessage)

		if jsonErr != nil {
			log.Print("JSON parsing error")
			log.Fatal(jsonErr)
		}

		log.Println("Parsed the following message")
		log.Printf("%+v\n", receivedMessage)

		// Respond to FB with receipt acknowledgement, process a response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Received okay."))
		log.Print("Acknowledged message receipt.")

		for _, entry := range receivedMessage.Entry {
			// Pass first messaging object directly as a webook event will only contain 1
			// https://developers.facebook.com/docs/messenger-platform/reference/webhook-events

			// Add the message to our queue for processing
			messageChan <- entry.Messaging[0]
			log.Println("Added message to queue for processing.")
		}
	}
}

func messageProcessor(messageChan <-chan facebookMessage) {

	// Will keep the go routine live as long as the channel is open
	for message := range messageChan {
		responseMsg := facebookMessage{MessagingType: "RESPONSE"}
		responseMsg.Message.Text = message.Message.Text
		responseMsg.Recipient.ID = message.Sender.ID

		go sendResponseMessage(responseMsg)
	}
}

func sendResponseMessage(responseMsg facebookMessage) {
	responseURL, err := url.Parse("https://graph.facebook.com/v5.0/me/messages")

	if err != nil {
		log.Panic(err)
	}

	// Add access token as query param
	queryParams := url.Values{}
	queryParams.Add("access_token", pageAccessToken)
	responseURL.RawQuery = queryParams.Encode()

	jsonResponseMsg, _ := json.Marshal(responseMsg)

	log.Println("Returning following JSON")
	log.Printf("%+v\n", responseMsg)

	resp, err := http.Post(
		responseURL.String(),
		"application/json",
		bytes.NewBuffer(jsonResponseMsg),
	)

	if err != nil {
		log.Panic("Got an unexpected response on sending FB message", err)
	}

	log.Println("Received following acknowledgement from FB messenger")
	log.Print(resp)
}
