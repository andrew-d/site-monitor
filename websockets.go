package main

import (
	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/gocraft/web"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func RegisterWebsockets(router *web.Router, ch chan interface{}) {
	wsBroadcaster := NewBroadcaster()

	router.Get("/ws", func(ctx *GlobalContext, w web.ResponseWriter, r *web.Request) {
		// The underlying struct is a web.AppResponseWriter.  We convert to this,
		// and then to the underlying http.ResponseWriter
		aw := w.(*web.AppResponseWriter)
		conn, err := upgrader.Upgrade(aw.ResponseWriter, r.Request, nil)
		if err != nil {
			log.WithFields(logrus.Fields{
				"err":       err,
				"requestId": ctx.RequestID,
			}).Error("Error upgrading websocket connection")
			return
		}

		// Start a new handler for this connection.
		go handleWebsocket(conn, wsBroadcaster.Listen())
	})

	// Read off the channel and push onto the broadcaster
	// TODO: push broadcaster up a level?
	go func() {
		for {
			msg, ok := <-ch
			if !ok {
				return
			}

			wsBroadcaster.Write(msg)
		}
	}()
}

func handleWebsocket(conn *websocket.Conn, l *BroadcastListener) {
	var err error
	var b []byte

	log.Info("Starting new websocket connection")
	defer l.Close()

	for {
		select {
		case msg, ok := <-l.Chan():
			if !ok {
				return
			}

			if b, err = json.Marshal(msg); err != nil {
				// TODO: handle me
				continue
			}

			if err = conn.WriteMessage(websocket.TextMessage, b); err != nil {
				// TODO: handle me
				return
			}
		}
	}
}
