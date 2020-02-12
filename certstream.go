package certstream

import (
	"github.com/gorilla/websocket"
	"github.com/jmoiron/jsonq"
	"github.com/pkg/errors"
	"time"
)

func CertStreamEventStream(skipHeartbeats bool) (chan jsonq.JsonQuery, chan error) {
	outputStream := make(chan jsonq.JsonQuery)
	errStream := make(chan error)

	go func() {
		for {
			c, _, err := websocket.DefaultDialer.Dial("wss://certstream.calidog.io", nil)

			if err != nil {
				errStream <- errors.Wrap(err, "Error connecting to certstream! Sleeping a few seconds and reconnecting... ")
				time.Sleep(5 * time.Second)
				continue
			}

			defer c.Close()
			defer close(outputStream)

			go func() {
				for {
					time.Sleep(15 * time.Second)
					if err := c.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
						errStream <- errors.Wrap(err, "Error sending ping, terminating ping sender")
						return
					}
				}
			}()

			for {
				var v interface{}
				err = c.ReadJSON(&v)
				if err != nil {
					errStream <- errors.Wrap(err, "Error decoding json frame!")
					c.Close()
					break
				}

				jq := jsonq.NewQuery(v)

				res, err := jq.String("message_type")
				if err != nil {
					errStream <- errors.Wrap(err, "Could not create jq object. Malformed json input recieved. Skipping.")
					continue
				}

				if skipHeartbeats && res == "heartbeat" {
					continue
				}

				outputStream <- *jq
			}
		}
	}()

	return outputStream, errStream
}
