package service

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Глобальная история сообщений для тестов
var globalHistory = make([]string, 0)

// Обработчик клиентского соединения для тестов
func handleClient(conn *websocket.Conn) {
	defer conn.Close()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if err := conn.WriteMessage(messageType, p); err != nil {
			return
		}
	}
}

func setupTest(t *testing.T) *sync.WaitGroup {
	// Enable test mode
	setTestMode(true)

	// Reset global state
	mu.Lock()
	clients = make(map[*websocket.Conn]bool)
	mu.Unlock()

	globalMu.Lock()
	globalClients = make(map[*websocket.Conn]bool)
	globalHistory = make([]string, 0)
	globalMu.Unlock()

	// Start message handlers
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		handleMessages()
	}()
	go func() {
		defer wg.Done()
		handleGlobalMessages()
	}()

	return &wg
}

func teardownTest(t *testing.T, wg *sync.WaitGroup) {
	cleanupTest()
	wg.Wait() // Wait for handlers to finish
	setTestMode(false)
}

func TestWebSocketMessageBroadcast(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	// Connect two clients
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Give the server time to process the connections
	time.Sleep(50 * time.Millisecond)

	// Send message from first client
	testMsg := "Hello, World!"
	err = ws1.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)

	// Wait for message to be broadcast
	time.Sleep(50 * time.Millisecond)

	// Read message from second client
	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(msg))
}

func TestGlobalWebSocketMessageBroadcast(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
	defer server.Close()

	// Connect two clients
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Give the server time to process the connections
	time.Sleep(50 * time.Millisecond)

	// Send message from first client
	testMsg := "Hello, Global!"
	err = ws1.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)

	// Wait for message to be broadcast
	time.Sleep(50 * time.Millisecond)

	// Read message from second client
	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(msg))
}

func TestWebSocketConnectionError(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create server that rejects WebSocket connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	// Try to connect
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.Error(t, err, "Expected connection error")
}

func TestWebSocketReadError(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Закрываем соединение для создания ошибки чтения
	ws.Close()

	// Даем время обработчику обнаружить закрытие
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	assert.Equal(t, 0, len(clients), "Client should be removed on error")
	mu.Unlock()
}

func TestInvalidWebSocketMessage(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Отправляем бинарное сообщение вместо текстового
	err = ws.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02, 0x03})
	require.NoError(t, err)

	// Проверяем, что соединение не разорвано
	err = ws.WriteMessage(websocket.TextMessage, []byte("ping"))
	require.NoError(t, err)
}

func TestLargeMessage(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Генерируем большое сообщение (100KB)
	largeMsg := strings.Repeat("a", 100*1024)
	err = ws1.WriteMessage(websocket.TextMessage, []byte(largeMsg))
	require.NoError(t, err)

	// Проверяем получение
	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, largeMsg, string(msg))
}

func TestGlobalChatMessageOrder(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Подключаем два клиента
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Отправляем несколько сообщений
	messages := []string{"first", "second", "third"}
	for _, msg := range messages {
		err = ws1.WriteMessage(websocket.TextMessage, []byte(msg))
		require.NoError(t, err)
	}

	// Проверяем порядок получения
	for _, expected := range messages {
		_, msg, err := ws2.ReadMessage()
		require.NoError(t, err)
		assert.Equal(t, expected, string(msg))
	}
}

func TestServerShutdownWithActiveConnections(t *testing.T) {
	wg := setupTest(t)

	// Create a channel to coordinate shutdown
	done := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create active connection
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Start a goroutine to read messages
	go func() {
		defer close(done)
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Give time for connection to establish
	time.Sleep(50 * time.Millisecond)

	// Close server and cleanup
	server.Close()
	teardownTest(t, wg)

	// Force close the connection
	ws.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
	ws.Close()

	// Wait for read loop to detect closure
	<-done

	// Try to write to closed connection
	err = ws.WriteMessage(websocket.TextMessage, []byte("test"))
	require.Error(t, err, "Expected error writing to closed connection")
}

func TestPingPong(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			HandshakeTimeout: 5 * time.Second,
			CheckOrigin:      func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		// Устанавливаем обработчик Pong
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			return nil
		})

		handleClient(conn)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Устанавливаем обработчик Pong
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(10 * time.Second))
		return nil
	})

	// Отправляем Ping
	err = ws.WriteMessage(websocket.PingMessage, []byte("ping"))
	require.NoError(t, err)

	// Проверяем, что соединение активно
	err = ws.WriteMessage(websocket.TextMessage, []byte("test"))
	require.NoError(t, err)
}

func TestMaxMessageSize(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create server with small message size limit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Set a small read limit
		conn.SetReadLimit(10) // Very small limit to ensure message is too big

		// Read one message and expect it to fail
		_, _, err = conn.ReadMessage()
		if err == nil {
			t.Error("Expected read limit exceeded error")
		}

		// Send close message
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "message too big"),
			time.Now().Add(time.Second))
	}))
	defer server.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Send message larger than limit
	largeMsg := strings.Repeat("a", 100)
	err = ws.WriteMessage(websocket.TextMessage, []byte(largeMsg))
	require.NoError(t, err)

	// Try to read response - should fail with close message
	_, _, err = ws.ReadMessage()
	require.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseMessageTooBig),
		"Expected CloseMessageTooBig error, got: %v", err)
}

func TestConcurrentConnections(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create multiple concurrent connections
	numClients := 5
	clients := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer ws.Close()
		clients[i] = ws
	}

	// Send messages from each client
	for i, ws := range clients {
		msg := fmt.Sprintf("Message from client %d", i)
		err := ws.WriteMessage(websocket.TextMessage, []byte(msg))
		require.NoError(t, err)
	}

	// Each client should receive messages from all other clients
	for _, ws := range clients {
		for i := 0; i < numClients; i++ {
			_, msg, err := ws.ReadMessage()
			require.NoError(t, err)
			assert.Contains(t, string(msg), "Message from client")
		}
	}
}

func TestWebSocketReconnection(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// First connection
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Close first connection
	ws1.Close()

	// Reconnect
	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Should be able to send/receive messages on new connection
	testMsg := "Test after reconnection"
	err = ws2.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)
}

func TestWebSocketBroadcastToOthers(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect three clients
	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	ws3, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws3.Close()

	// Give time for connections to establish
	time.Sleep(50 * time.Millisecond)

	// Send message from first client
	testMsg := "Broadcast test"
	err = ws1.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)

	// Second and third clients should receive the message
	_, msg2, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(msg2))

	_, msg3, err := ws3.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(msg3))
}

func TestWebSocketCloseHandling(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Send close frame
	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)

	// Try to send message after close
	err = ws.WriteMessage(websocket.TextMessage, []byte("test"))
	assert.Error(t, err)

	ws.Close()
}

func TestWebSocketPingPongTimeout(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect with custom dialer that has short timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: 1 * time.Second,
	}

	ws, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Set very short read deadline
	ws.SetReadDeadline(time.Now().Add(10 * time.Millisecond))

	// Try to read - should timeout
	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
	netErr, ok := err.(net.Error)
	assert.True(t, ok, "Expected net.Error")
	assert.True(t, netErr.Timeout(), "Expected timeout error")
}

func TestWebSocketBinaryMessageHandling(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Read one message
		messageType, _, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// If binary message received, close connection
		if messageType == websocket.BinaryMessage {
			conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "binary messages not supported"),
				time.Now().Add(time.Second))
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Send binary message
	binaryData := []byte{0x01, 0x02, 0x03}
	err = ws.WriteMessage(websocket.BinaryMessage, binaryData)
	require.NoError(t, err)

	// Try to read response - should get close message
	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseUnsupportedData),
		"Expected CloseUnsupportedData error, got: %v", err)
}

func TestWebSocketInvalidUpgrade(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send invalid response to prevent upgrade
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request"))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Try to connect - should fail
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.Error(t, err)
}

func TestWebSocketConcurrentBroadcast(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create multiple clients
	numClients := 10
	clients := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer ws.Close()
		clients[i] = ws
	}

	// Send messages concurrently
	var sendWg sync.WaitGroup
	for i := 0; i < numClients; i++ {
		sendWg.Add(1)
		go func(idx int) {
			defer sendWg.Done()
			msg := fmt.Sprintf("Message %d", idx)
			err := clients[idx].WriteMessage(websocket.TextMessage, []byte(msg))
			assert.NoError(t, err)
		}(i)
	}

	// Wait for all messages to be sent
	sendWg.Wait()

	// Each client should receive all messages
	for _, ws := range clients {
		for i := 0; i < numClients; i++ {
			_, msg, err := ws.ReadMessage()
			require.NoError(t, err)
			assert.Contains(t, string(msg), "Message")
		}
	}
}

func TestWebSocketCloseOnError(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Force an error by closing the connection immediately
		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Try to read - should fail due to closed connection
	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseAbnormalClosure))
}

func TestWebSocketMessageValidation(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		// Read message
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		// Validate message length
		if len(msg) > 1024 {
			conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "message too large"),
				time.Now().Add(time.Second))
			return
		}

		// Echo valid message
		conn.WriteMessage(websocket.TextMessage, msg)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Send large message
	largeMsg := strings.Repeat("a", 2048)
	err = ws.WriteMessage(websocket.TextMessage, []byte(largeMsg))
	require.NoError(t, err)

	// Should receive close message
	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseMessageTooBig))

	// Create new connection
	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	// Send valid message
	validMsg := "Hello"
	err = ws2.WriteMessage(websocket.TextMessage, []byte(validMsg))
	require.NoError(t, err)

	// Should receive echo
	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, validMsg, string(msg))
}

func TestWebSocketHandlerErrors(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handleConnections))
		defer server.Close()

		// Try to connect with POST method
		resp, err := http.Post(strings.Replace(server.URL, "http", "ws", 1), "", nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid WebSocket Protocol", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handleConnections))
		defer server.Close()

		// Try to connect without WebSocket protocol
		resp, err := http.Get(server.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestGlobalWebSocketHandlerErrors(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
		defer server.Close()

		// Try to connect with POST method
		resp, err := http.Post(strings.Replace(server.URL, "http", "ws", 1), "", nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid WebSocket Protocol", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
		defer server.Close()

		// Try to connect without WebSocket protocol
		resp, err := http.Get(server.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestWebSocketConcurrentConnections(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Create multiple concurrent connections
	var connWg sync.WaitGroup
	numConns := 50
	for i := 0; i < numConns; i++ {
		connWg.Add(1)
		go func(idx int) {
			defer connWg.Done()

			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			require.NoError(t, err)
			defer ws.Close()

			// Send a message
			msg := fmt.Sprintf("Message from connection %d", idx)
			err = ws.WriteMessage(websocket.TextMessage, []byte(msg))
			require.NoError(t, err)

			// Read response
			_, _, err = ws.ReadMessage()
			require.NoError(t, err)
		}(i)
	}

	connWg.Wait()
}

func TestWebSocketMessageTypes(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	testCases := []struct {
		name        string
		messageType int
		data        []byte
		expectError bool
	}{
		{
			name:        "Text Message",
			messageType: websocket.TextMessage,
			data:        []byte("Hello"),
			expectError: false,
		},
		{
			name:        "Binary Message",
			messageType: websocket.BinaryMessage,
			data:        []byte{1, 2, 3},
			expectError: true,
		},
		{
			name:        "Ping Message",
			messageType: websocket.PingMessage,
			data:        []byte("ping"),
			expectError: false,
		},
		{
			name:        "Pong Message",
			messageType: websocket.PongMessage,
			data:        []byte("pong"),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ws.WriteMessage(tc.messageType, tc.data)
			require.NoError(t, err)

			if tc.expectError {
				_, _, err = ws.ReadMessage()
				assert.Error(t, err)
			} else if tc.messageType == websocket.TextMessage {
				_, msg, err := ws.ReadMessage()
				require.NoError(t, err)
				assert.Equal(t, string(tc.data), string(msg))
			}
		})
	}
}

func TestWebSocketConnectionLimit(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Try to create more connections than the limit
	maxConns := 1000
	conns := make([]*websocket.Conn, 0, maxConns)
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	for i := 0; i < maxConns; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			// If we get an error, we've hit the connection limit
			assert.Contains(t, err.Error(), "connection")
			break
		}
		conns = append(conns, ws)
	}

	// We should be able to create at least some connections
	assert.Greater(t, len(conns), 0)
	assert.Less(t, len(conns), maxConns)
}
