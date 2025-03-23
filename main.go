package main

import (
    "fmt"
    "net/http"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    _ "github.com/mattn/go-sqlite3"
)

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

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        phoneNumber := r.URL.Query().Get("phone") // URL se phone number lo
        if phoneNumber == "" {
            fmt.Fprintf(w, "Phone number daal: /?phone=918516847084")
            return
        }
        if client.Store.ID == nil {
            pairCode, err := client.PairPhone(phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
            if err != nil {
                fmt.Fprintf(w, "Error generating pair code: %v", err)
                return
            }
            fmt.Fprintf(w, "Pairing Code: %s\nWhatsApp pe daalo: Settings > Linked Devices > Link with phone number", pairCode)
            client.Connect()
        } else {
            fmt.Fprintf(w, "Already Connected!")
        }
    })

    fmt.Println("Starting web server on :8080...")
    http.ListenAndServe(":8080", nil)
}
