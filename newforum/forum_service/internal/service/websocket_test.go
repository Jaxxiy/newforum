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

var globalHistory = make([]string, 0)

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
	setTestMode(true)

	mu.Lock()
	clients = make(map[*websocket.Conn]bool)
	mu.Unlock()

	globalMu.Lock()
	globalClients = make(map[*websocket.Conn]bool)
	globalHistory = make([]string, 0)
	globalMu.Unlock()

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
	wg.Wait()
	setTestMode(false)
}

func TestWebSocketMessageBroadcast(t *testing.T) {
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

	time.Sleep(50 * time.Millisecond)

	testMsg := "Hello, World!"
	err = ws1.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(msg))
}

func TestGlobalWebSocketMessageBroadcast(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	time.Sleep(50 * time.Millisecond)

	testMsg := "Hello, Global!"
	err = ws1.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, testMsg, string(msg))
}

func TestWebSocketConnectionError(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

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

	ws.Close()

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

	err = ws.WriteMessage(websocket.BinaryMessage, []byte{0x01, 0x02, 0x03})
	require.NoError(t, err)

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

	largeMsg := strings.Repeat("a", 100*1024)
	err = ws1.WriteMessage(websocket.TextMessage, []byte(largeMsg))
	require.NoError(t, err)

	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, largeMsg, string(msg))
}

func TestServerShutdownWithActiveConnections(t *testing.T) {
	wg := setupTest(t)

	done := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	go func() {
		defer close(done)
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)

	server.Close()
	teardownTest(t, wg)

	ws.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
	ws.Close()

	<-done

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

	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(10 * time.Second))
		return nil
	})

	err = ws.WriteMessage(websocket.PingMessage, []byte("ping"))
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.TextMessage, []byte("test"))
	require.NoError(t, err)
}

func TestMaxMessageSize(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer conn.Close()

		conn.SetReadLimit(10)

		_, _, err = conn.ReadMessage()
		if err == nil {
			t.Error("Expected read limit exceeded error")
		}

		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "message too big"),
			time.Now().Add(time.Second))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	largeMsg := strings.Repeat("a", 100)
	err = ws.WriteMessage(websocket.TextMessage, []byte(largeMsg))
	require.NoError(t, err)

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

	numClients := 5
	clients := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer ws.Close()
		clients[i] = ws
	}

	for i, ws := range clients {
		msg := fmt.Sprintf("Message from client %d", i)
		err := ws.WriteMessage(websocket.TextMessage, []byte(msg))
		require.NoError(t, err)
	}

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

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

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

	ws1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws1.Close()

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	ws3, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws3.Close()

	time.Sleep(50 * time.Millisecond)

	testMsg := "Broadcast test"
	err = ws1.WriteMessage(websocket.TextMessage, []byte(testMsg))
	require.NoError(t, err)

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

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	err = ws.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	require.NoError(t, err)

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

	dialer := websocket.Dialer{
		HandshakeTimeout: 1 * time.Second,
	}

	ws, _, err := dialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	ws.SetReadDeadline(time.Now().Add(10 * time.Millisecond))

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

		messageType, _, err := conn.ReadMessage()
		if err != nil {
			return
		}

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

	binaryData := []byte{0x01, 0x02, 0x03}
	err = ws.WriteMessage(websocket.BinaryMessage, binaryData)
	require.NoError(t, err)

	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseUnsupportedData),
		"Expected CloseUnsupportedData error, got: %v", err)
}

func TestWebSocketInvalidUpgrade(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request"))
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	assert.Error(t, err)
}

func TestWebSocketConcurrentBroadcast(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	numClients := 10
	clients := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer ws.Close()
		clients[i] = ws
	}

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

	sendWg.Wait()

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

		conn.Close()
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

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

		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if len(msg) > 1024 {
			conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseMessageTooBig, "message too large"),
				time.Now().Add(time.Second))
			return
		}

		conn.WriteMessage(websocket.TextMessage, msg)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	largeMsg := strings.Repeat("a", 2048)
	err = ws.WriteMessage(websocket.TextMessage, []byte(largeMsg))
	require.NoError(t, err)

	_, _, err = ws.ReadMessage()
	assert.Error(t, err)
	assert.True(t, websocket.IsCloseError(err, websocket.CloseMessageTooBig))

	ws2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws2.Close()

	validMsg := "Hello"
	err = ws2.WriteMessage(websocket.TextMessage, []byte(validMsg))
	require.NoError(t, err)

	_, msg, err := ws2.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, validMsg, string(msg))
}

func TestWebSocketHandlerErrors(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	t.Run("Invalid HTTP Method", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handleConnections(w, r)
		}))
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL, nil)
		require.NoError(t, err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid WebSocket Protocol", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handleConnections))
		defer server.Close()

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
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handleGlobalConnections(w, r)
		}))
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL, nil)
		require.NoError(t, err)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid WebSocket Protocol", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
		defer server.Close()

		resp, err := http.Get(server.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
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
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ws.WriteMessage(tc.messageType, tc.data)
			require.NoError(t, err)

			if !tc.expectError {
				_, msg, err := ws.ReadMessage()
				require.NoError(t, err)
				if tc.messageType == websocket.TextMessage {
					assert.Equal(t, string(tc.data), string(msg))
				}
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

	maxConns := 50
	conns := make([]*websocket.Conn, 0, maxConns)
	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	var connError error
	for i := 0; i < maxConns; i++ {
		ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			connError = err
			break
		}
		conns = append(conns, ws)

		err = ws.WriteMessage(websocket.TextMessage, []byte("test"))
		if err != nil {
			connError = err
			break
		}
	}

	assert.Greater(t, len(conns), 0)

	if connError != nil {
		assert.Contains(t, connError.Error(), "connection")
	}

	for _, conn := range conns {
		if conn != nil {
			err := conn.WriteMessage(websocket.TextMessage, []byte("test"))
			assert.NoError(t, err)
		}
	}
}
