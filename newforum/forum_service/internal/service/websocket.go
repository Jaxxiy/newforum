package service

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Общие переменные (для первого соединения)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Разрешить все источники (не безопасно для продакшена)
	},
}

var (
	clients   = make(map[*websocket.Conn]bool) // Для первого соединения
	broadcast = make(chan string)              // Для первого соединения
	mu        sync.RWMutex                     // Для первого соединения
)

// Новые переменные для второго соединения (глобального чата)
var upgraderGlobal = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Разрешить все источники (не безопасно для продакшена)
	},
}

var (
	globalClients   = make(map[*websocket.Conn]bool) // Для второго соединения
	globalBroadcast = make(chan string)              // Для второго соединения
	globalMu        sync.RWMutex                     // Для второго соединения
)

// For testing purposes
var (
	isTestMode = false
	testDone   = make(chan struct{})
	ready      = make(chan struct{})
)

func StartWebSocket() {
	//Первое соединение
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()
	log.Println("WebSocket server (первое соединение) started on :8081")

	//Второе соединение
	http.HandleFunc("/ws/global", handleGlobalConnections)               // Новый обработчик
	go handleGlobalMessages()                                            // Новая горутина
	log.Println("WebSocket server (второе соединение) started on :8082") //Другой порт

	// Signal that handlers are ready
	if isTestMode {
		close(ready)
	}

	// Запускаем сервер
	go func() {
		if err := http.ListenAndServe(":8081", nil); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server (первое соединение) error: %v", err)
		}
	}()

	if err := http.ListenAndServe(":8082", nil); err != nil && err != http.ErrServerClosed {
		log.Printf("WebSocket server (второе соединение) error: %v", err)
	}
}

// Обработчик для первого соединения
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	mu.Lock()
	clients[ws] = true
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(clients, ws)
		mu.Unlock()
		ws.Close()
	}()

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		select {
		case broadcast <- string(msg):
		default:
			log.Println("Message dropped: channel full")
		}

		if isTestMode {
			select {
			case <-testDone:
				return
			default:
			}
		}
	}
}

// Обработчик для второго соединения (глобального чата)
func handleGlobalConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgraderGlobal.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	globalMu.Lock()
	globalClients[ws] = true
	globalMu.Unlock()

	defer func() {
		globalMu.Lock()
		delete(globalClients, ws)
		globalMu.Unlock()
		ws.Close()
	}()

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		select {
		case globalBroadcast <- string(msg):
		default:
			log.Println("Global message dropped: channel full")
		}

		if isTestMode {
			select {
			case <-testDone:
				return
			default:
			}
		}
	}
}

// Обработчик сообщений для первого соединения
func handleMessages() {
	for {
		select {
		case msg := <-broadcast:
			mu.RLock()
			for client := range clients {
				err := client.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					mu.RUnlock()
					mu.Lock()
					client.Close()
					delete(clients, client)
					mu.Unlock()
					mu.RLock()
					continue
				}
			}
			mu.RUnlock()
		case <-testDone:
			if isTestMode {
				return
			}
		}
	}
}

// Обработчик сообщений для второго соединения (глобального чата)
func handleGlobalMessages() {
	for {
		select {
		case msg := <-globalBroadcast:
			globalMu.RLock()
			for client := range globalClients {
				err := client.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					globalMu.RUnlock()
					globalMu.Lock()
					client.Close()
					delete(globalClients, client)
					globalMu.Unlock()
					globalMu.RLock()
					continue
				}
			}
			globalMu.RUnlock()
		case <-testDone:
			if isTestMode {
				return
			}
		}
	}
}

// For testing purposes
func setTestMode(enabled bool) {
	isTestMode = enabled
	if enabled {
		testDone = make(chan struct{})
		ready = make(chan struct{})
		broadcast = make(chan string, 10)       // Buffered channel for tests
		globalBroadcast = make(chan string, 10) // Buffered channel for tests
	}
}

func cleanupTest() {
	if isTestMode {
		close(testDone)
	}

	mu.Lock()
	for client := range clients {
		client.Close()
		delete(clients, client)
	}
	mu.Unlock()

	globalMu.Lock()
	for client := range globalClients {
		client.Close()
		delete(globalClients, client)
	}
	globalMu.Unlock()
}

// WaitForReady waits for handlers to be ready in test mode
func WaitForReady() {
	if isTestMode {
		<-ready
	}
}
