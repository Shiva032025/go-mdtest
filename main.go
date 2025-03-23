package main

import (
    "fmt"
    "net/http"
    "time"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    _ "github.com/mattn/go-sqlite3"
)

var lastPairTime time.Time

func main() {
    db, err := sqlstore.New("sqlite3", "file:mdtest.db?_foreign_keys=on", nil)
    if err != nil {
        panic(err)
    }
    deviceStore, err := db.GetFirstDevice()
    if err != nil {
        panic(err)
    }
    client := whatsmeow.NewClient(deviceStore, nil)

    // Event logging for debugging
    client.AddEventHandler(func(evt interface{}) {
        fmt.Printf("Event: %v\n", evt)
    })

    err = client.Connect()
    if err != nil {
        fmt.Printf("Failed to connect: %v\n", err)
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        phoneNumber := r.URL.Query().Get("phone")
        if phoneNumber == "" {
            fmt.Fprintf(w, "Phone number daal: /?phone=918516847084")
            return
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
                time.Sleep(2 * time.Second) // Mimic human delay
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
