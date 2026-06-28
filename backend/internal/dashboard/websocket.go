package dashboard

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/config"
	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// CORS header
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleLiveTail(logger *slog.Logger, broker *transport.NatsBroker, config *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// upgrade HTTP to WebSocket
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("Failed to upgrade websocket", "error", err)
			return
		}
		defer ws.Close()

		// create a cancellable context tied to this WebSocket connection
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()

		// listen for client disconnects to cancel the context
		go func() {
			for {
				// if ReadMessage fails, the client disconnected or the connection died
				if _, _, err := ws.ReadMessage(); err != nil {
					logger.Debug("Websocket client disconnected")
					cancel()
					return
				}
			}
		}()

		// start tailing NATS
		liveTailCh, cleanup, err := broker.TailLiveLogs(ctx, transport.LiveTailSubject, config.LiveTailMaxBatch)
		if err != nil {
			logger.Error("Failed to start NATS tail", "error", err)
			ws.WriteMessage(websocket.CloseMessage, []byte("Internal Server Error"))
			return
		}
		defer cleanup() // ensure NATS consumer stops when the websocket closes

		ticker := time.NewTicker(time.Duration(config.LiveTailRefreshInterval) * time.Millisecond) // default flush interval: 500ms
		defer ticker.Stop()

		var batch []byte // accumulate payloads between ticks

		// pump NATS messages to the WebSocket
		for {
			select {
			case <-ctx.Done():
				// context cancelled (client disconnected or server shutting down)
				return
			case payload, ok := <-liveTailCh:
				if !ok {
					// channel closed
					return
				}

				// append raw messagePack bytes directly
				batch = append(batch, payload...)

			case <-ticker.C:
				if len(batch) == 0 {
					continue
				}

				// write the binary payload directly to the WebSocket
				err = ws.WriteMessage(websocket.BinaryMessage, batch)
				if err != nil {
					logger.Error("Failed to write to websocket", "error", err)
					return // exiting the loop triggers defer cleanup() and cancel()
				}
				batch = batch[:0]
			}
		}
	}
}
