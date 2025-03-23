package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    "net/http"
    "bytes"
    "encoding/json"

    "go.mau.fi/whatsmeow"
    "go.mau.fi/whatsmeow/store/sqlstore"
    "go.mau.fi/whatsmeow/types"
    "go.mau.fi/whatsmeow/types/events"
    waProto "go.mau.fi/whatsmeow/proto/waE2E"
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

    fmt.Println("Starting WhatsApp client...")

    if client.Store.ID == nil {
        if len(os.Args) < 2 {
            fmt.Println("Phone number provide karo (jaise: ./mdtest 919876543210)")
            return
        }
        phoneNumber := os.Args[1]
        fmt.Println("Pairing phone:", phoneNumber)

        // Pehle connect try karo
        err = client.Connect()
        if err != nil {
            fmt.Println("Initial connection error:", err)
            return
        }
        time.Sleep(5 * time.Second) // Connection stabilize hone ka wait

        pairCode, err := client.PairPhone(phoneNumber, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
        if err != nil {
            fmt.Println("Pairing error:", err)
            return
        }
        fmt.Println("Pairing code:", pairCode)
        fmt.Println("WhatsApp pe jao: Settings > Linked Devices > Link with phone number, aur code daalo.")
        time.Sleep(10 * time.Second) // Pairing complete hone ka wait
    } else {
        err = client.Connect()
        if err != nil {
            fmt.Println("Connection error:", err)
            return
        }
    }

    fmt.Println("Connected to WhatsApp!")
    client.AddEventHandler(func(evt interface{}) {
        switch v := evt.(type) {
        case *events.Message:
            msg := v.Message.GetConversation()
            from := v.Info.Sender.String()
            fmt.Println("Received from", from, ":", msg)
            data := map[string]string{
                "from":    from,
                "message": msg,
            }
            jsonData, _ := json.Marshal(data)
            resp, err := http.Post("https://yourdomain.com/webhook.php", "application/json", bytes.NewBuffer(jsonData))
            if err != nil {
                fmt.Println("Webhook error:", err)
            } else {
                resp.Body.Close()
            }
        }
    })

    if len(os.Args) > 2 && os.Args[1] == "send" {
        jid, err := types.ParseJID(os.Args[2])
        if err != nil {
            fmt.Println("Invalid JID:", err)
            return
        }
        text := os.Args[3]
        msg := &waProto.Message{
            Conversation: &text,
        }
        _, err = client.SendMessage(context.Background(), jid, msg)
        if err != nil {
            fmt.Println("Send error:", err)
        } else {
            fmt.Println("Message sent!")
        }
    }

    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
    client.Disconnect()
}
