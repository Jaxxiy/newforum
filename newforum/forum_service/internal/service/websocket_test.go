package service

import (
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

func setupTest(t *testing.T) *sync.WaitGroup {
	// Enable test mode
	setTestMode(true)

	// Reset global state
	mu.Lock()
	clients = make(map[*websocket.Conn]bool)
	mu.Unlock()

	globalMu.Lock()
	globalClients = make(map[*websocket.Conn]bool)
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

func TestWebSocketConnection(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Give the server time to process the connection
	time.Sleep(50 * time.Millisecond)

	// Verify client was added
	mu.RLock()
	assert.True(t, clients[ws], "Client should be added to clients map")
	clientCount := len(clients)
	mu.RUnlock()
	assert.Equal(t, 1, clientCount, "Should have exactly one client")
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

func TestGlobalWebSocketConnection(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect to WebSocket server
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer ws.Close()

	// Give the server time to process the connection
	time.Sleep(50 * time.Millisecond)

	// Verify client was added
	globalMu.RLock()
	assert.True(t, globalClients[ws], "Client should be added to global clients map")
	clientCount := len(globalClients)
	globalMu.RUnlock()
	assert.Equal(t, 1, clientCount, "Should have exactly one global client")
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

func TestWebSocketClientDisconnection(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handleConnections))
	defer server.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Give the server time to process the connection
	time.Sleep(50 * time.Millisecond)

	// Verify client was added
	mu.RLock()
	assert.True(t, clients[ws], "Client should be added to clients map")
	mu.RUnlock()

	// Disconnect client
	ws.Close()

	// Wait for cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify client was removed
	mu.RLock()
	assert.False(t, clients[ws], "Client should be removed from clients map")
	mu.RUnlock()
}

func TestGlobalWebSocketClientDisconnection(t *testing.T) {
	wg := setupTest(t)
	defer teardownTest(t, wg)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handleGlobalConnections))
	defer server.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	// Give the server time to process the connection
	time.Sleep(50 * time.Millisecond)

	// Verify client was added
	globalMu.RLock()
	assert.True(t, globalClients[ws], "Client should be added to global clients map")
	globalMu.RUnlock()

	// Disconnect client
	ws.Close()

	// Wait for cleanup
	time.Sleep(50 * time.Millisecond)

	// Verify client was removed
	globalMu.RLock()
	assert.False(t, globalClients[ws], "Client should be removed from global clients map")
	globalMu.RUnlock()
}
