package main

import (
	"embed"
	"io/ioutil"
	stdLog "log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed web
var webAssets embed.FS

var wsUpgrader = websocket.Upgrader{
	HandshakeTimeout: 5 * time.Second,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return strings.HasPrefix(origin, "http://127.0.0.1:") ||
			strings.HasPrefix(origin, "http://localhost:") ||
			origin == "http://127.0.0.1" ||
			origin == "http://localhost"
	},
}

type wsClient struct {
	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	closeOnce sync.Once
}

func (c *wsClient) shutdown() {
	c.closeOnce.Do(func() {
		close(c.done)
		c.conn.Close()
	})
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		c.shutdown()
	}()
	for {
		select {
		case data := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *wsClient) readPump(h *wsHub) {
	defer func() {
		h.remove(c)
		c.shutdown()
	}()
	c.conn.SetReadLimit(1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

type wsHub struct {
	mu      sync.Mutex
	clients map[*wsClient]struct{}
	latest  []byte
}

var uiHub = &wsHub{clients: map[*wsClient]struct{}{}}

func (h *wsHub) add(c *wsClient) []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	return h.latest
}

func (h *wsHub) remove(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *wsHub) broadcast(data []byte) {
	h.mu.Lock()
	h.latest = append(h.latest[:0], data...)
	clients := make([]*wsClient, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()
	for _, c := range clients {
		select {
		case c.send <- data:
		default:
			// A slow client misses this frame; the next update supersedes it.
		}
	}
}

func (h *wsHub) handleWS(c echo.Context) error {
	conn, err := wsUpgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	client := &wsClient{
		conn: conn,
		send: make(chan []byte, 8),
		done: make(chan struct{}),
	}
	latest := h.add(client)
	if latest == nil {
		latest = uiPub.currentJSON()
	}
	go client.writePump()
	select {
	case client.send <- latest:
	default:
	}
	client.readPump(h)
	return nil
}

func serveWebAsset(c echo.Context, name, contentType string) error {
	data, err := webAssets.ReadFile(name)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	return c.Blob(http.StatusOK, contentType, data)
}

func runUIServer(addr string) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.StdLogger = stdLog.New(ioutil.Discard, "", 0)
	e.Logger.SetOutput(ioutil.Discard)
	e.Use(middleware.Recover())

	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusFound, "/ui")
	})
	e.GET("/ui", func(c echo.Context) error {
		return serveWebAsset(c, "web/index.html", "text/html; charset=utf-8")
	})
	e.GET("/style.css", func(c echo.Context) error {
		return serveWebAsset(c, "web/style.css", "text/css; charset=utf-8")
	})
	e.GET("/app.js", func(c echo.Context) error {
		return serveWebAsset(c, "web/app.js", "application/javascript; charset=utf-8")
	})
	e.GET("/api/state", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "no-store")
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		return c.Blob(http.StatusOK, "application/json; charset=utf-8", uiPub.currentJSON())
	})
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]bool{"ok": true})
	})
	e.GET("/ws", uiHub.handleWS)

	return e.Start(addr)
}
