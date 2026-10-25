package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Events streams domain events over Server-Sent Events. Each frame is an
// unnamed `data:` line carrying {"type":"task|attempt|event","data":{...}}.
func (h *Handler) Events(c *fiber.Ctx) error {
	c.Set(fiber.HeaderContentType, "text/event-stream")
	c.Set(fiber.HeaderCacheControl, "no-cache")
	c.Set(fiber.HeaderConnection, fiber.HeaderKeepAlive)
	c.Set("X-Accel-Buffering", "no")

	events := h.Engine.Subscribe()
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer h.Engine.Unsubscribe(events)

		write := func(payload []byte) bool {
			if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
				return false
			}
			return w.Flush() == nil
		}

		if !write([]byte(`{"type":"hello","data":null}`)) {
			return
		}
		heartbeat := time.NewTicker(25 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}
				payload, err := json.Marshal(event)
				if err != nil {
					continue
				}
				if !write(payload) {
					return
				}
			case <-heartbeat.C:
				if _, err := w.WriteString(": ping\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})
	return nil
}
