package main

import (
    "fmt"
    "net/http"
    "time"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    "go.mau.fi/whatsmeow/store"
    _ "github.com/mattn/go-sqlite3"
)

var lastPairTime time.Time
var clients = make(map[string]*whatsmeow.Client) // Map to store clients for each phone number

func main() {
    db, err := sqlstore.New("sqlite3", "file:mdtest.db?_foreign_keys=on", nil)
    if err != nil {
        panic(err)
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        phoneNumber := r.URL.Query().Get("phone")
        if phoneNumber == "" {
            fmt.Fprintf(w, "Phone number daal: /?phone=918516847084")
            return
        }

        // Check if client for this phone number already exists
        client, exists := clients[phoneNumber]
        if !exists {
            // Create a new device for this phone number
            deviceStore := store.Device{}
            client = whatsmeow.NewClient(&deviceStore, nil)
            clients[phoneNumber] = client

            // Save the device to the database
            err = db.PutDevice(&deviceStore)
            if err != nil {
                fmt.Fprintf(w, "Error saving device to database: %v", err)
                return
            }

            // Event logging
            client.AddEventHandler(func(evt interface{}) {
                fmt.Printf("Event for %s: %v\n", phoneNumber, evt)
            })

            err = client.Connect()
            if err != nil {
                fmt.Fprintf(w, "Error connecting: %v", err)
                return
            }
        }

        if client.Store.ID == nil {
            if time.Since(lastPairTime) < 5*time.Minute {
                fmt.Fprintf(w, "Please wait 5 minutes before generating a new pair code")
                return
            }
            if !client.IsConnected() {
                err := client.Connect()
                if err != nil {
                    fmt.Fprintf(w, "Error connecting: %v", err)
                    return
                }
                time.Sleep(2 * time.Second)
            }
            pairCode, err := client.PairPhone(phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Windows)")
            if err != nil {
                fmt.Fprintf(w, "Error generating pair code: %v", err)
                return
            }
            fmt.Fprintf(w, "Pairing Code: %s\nWhatsApp pe daalo: Settings > Linked Devices > Link with phone number", pairCode)
            lastPairTime = time.Now()
        } else {
            fmt.Fprintf(w, "Already Connected!")
        }
    })

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "OK")
    })

    fmt.Println("Starting web server on :10000...")
    http.ListenAndServe(":10000", nil)
}
