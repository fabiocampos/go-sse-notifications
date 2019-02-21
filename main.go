package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Notification struct {
	Message string `json:"message"`
}

var clients []chan []byte

func main() {
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/system/", http.StripPrefix("/system/", fs))

	http.HandleFunc("/notifications", HandleNotifications)
	http.HandleFunc("/", HandleSync)
	http.ListenAndServe(":3000", nil)

}

func HandleSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	closedConnection := w.(http.CloseNotifier).CloseNotify()

	messageChan := make(chan []byte)
	go func() {
		<-closedConnection
		fmt.Println("Removing client")
		for i, clientMessageChan := range clients {
			if messageChan == clientMessageChan {
				clients = append(clients[:i], clients[i+1:]...)
			}
		}
	}()

	fmt.Println("Appending client %s", len(clients))
	clients = append(clients, messageChan)
	for {
		notification := <-messageChan
		fmt.Fprintf(w, "data: %s\n\n", notification)
		fmt.Println("Receiving event %s", string(notification))
		flusher.Flush()
	}

}
func HandleNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var notification *Notification
	err := decoder.Decode(&notification)
	if err != nil {
		http.Error(w, "Bad data", http.StatusBadRequest)
		return
	}

	log.Println(notification.Message)
	go func() {
		for _, clientMessageChan := range clients {
			clientMessageChan <- []byte(notification.Message)
		}
	}()
}
