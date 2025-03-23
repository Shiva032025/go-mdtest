package main

import (
    "fmt"
    "net/http"
    "time"
    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    "go.mau.fi/whatsmeow/store"
    waLog "go.mau.fi/whatsmeow/util/log"
    _ "github.com/mattn/go-sqlite3"
)

var lastPairTime time.Time
var clients = make(map[string]*whatsmeow.Client) // Map to store clients for each phone number
var devices = make(map[string]*store.Device)     // Map to store devices for each phone number

func main() {
    // Use in-memory database to avoid file system issues on Render
    db, err := sqlstore.New("sqlite3", ":memory:", nil)
    if err != nil {
        fmt.Printf("Error initializing database: %v\n", err)
        return
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
            deviceStore := &store.Device{}
            // Use whatsmeow's built-in logger
            deviceStore.Log = waLog.Stdout("Device", "DEBUG", true)

            client = whatsmeow.NewClient(deviceStore, nil)
            if client == nil {
                fmt.Fprintf(w, "Error creating WhatsApp client")
                return
            }
            fmt.Printf("Created client for %s: %v\n", phoneNumber, client)
            clients[phoneNumber] = client
            devices[phoneNumber] = deviceStore // Store device in map

            // Event logging
            client.AddEventHandler(func(evt interface{}) {
                fmt.Printf("Event for %s: %v\n", phoneNumber, evt)
                // Save device to database after pairing (when JID is set)
                if client.Store.ID != nil {
                    err := db.PutDevice(deviceStore)
                    if err != nil {
                        fmt.Printf("Error saving device to database after pairing: %v\n", err)
                    }
                }
            })

            // Connect with error handling
            fmt.Printf("Attempting to connect for %s...\n", phoneNumber)
            err = client.Connect()
            if err != nil {
                fmt.Printf("Failed to connect for %s: %v\n", phoneNumber, err)
                fmt.Fprintf(w, "Error connecting to WhatsApp: %v", err)
                return
            }
            fmt.Printf("Connected successfully for %s\n", phoneNumber)
        }

        if client.Store.ID == nil {
            if time.Since(lastPairTime) < 5*time.Minute {
                fmt.Fprintf(w, "Please wait 5 minutes before generating a new pair code")
                return
            }
            if !client.IsConnected() {
                fmt.Printf("Reconnecting for %s...\n", phoneNumber)
                err := client.Connect()
                if err != nil {
                    fmt.Printf("Failed to reconnect for %s: %v\n", phoneNumber, err)
                    fmt.Fprintf(w, "Error connecting: %v", err)
                    return
                }
                time.Sleep(2 * time.Second)
            }
            fmt.Printf("Generating pair code for %s...\n", phoneNumber)
            pairCode, err := client.PairPhone(phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Windows)")
            if err != nil {
                fmt.Printf("Error generating pair code for %s: %v\n", phoneNumber, err)
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

    // Debug endpoint to check client status
    http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Active Clients:\n")
        for phone, client := range clients {
            status := "Not Connected"
            if client.IsConnected() {
                status = "Connected"
            }
            fmt.Fprintf(w, "Phone: %s, Status: %s, ID: %v\n", phone, status, client.Store.ID)
        }
    })

    fmt.Println("Starting web server on :10000...")
    err = http.ListenAndServe(":10000", nil)
    if err != nil {
        fmt.Printf("Error starting server: %v\n", err)
    }
}
