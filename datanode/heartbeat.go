package datanode

import (
	"bytes"
	"encoding/json"
	"log"
	"time"
	"net/http"
	"github.com/Rahul6700/Foodo/shared"
)

func StartHeartBeat(lbAddr, myApiAddr string){
	// hardcoding the DN's IP (the system running the DN)
	const IP_addr = "localhost"

	// this is the nodeID that we will send the loadb
	myURL := "http://" + IP_addr + myApiAddr // myApiAddr is the port on which the DN is running
	log.Printf("DN %s is sending heartbeat", myURL)

	// creating a new ticker obj that triggers every 5 seconds
	ticker := time.NewTicker(5*time.Second)

	for range ticker.C {
		// get the value from the automic counter
		load := ActiveWrites.Load()

		// create the req payload
		payload := shared.HeartbeatPayload {
			NodeID: myURL,
			ActiveWrites: int(load),
		}

		// convert it to JSON form
		jsonData, err := json.Marshal(payload)
		if err != nil {
			log.Printf("%s failed to marshal heartbeat: %s", myURL, err)
			continue // skip this ticker and continue from next
		}

		resp, err := http.Post(lbAddr+"/heartbeat", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			log.Printf("%s failed to send heartbeat", myURL)
			continue // skip this ticker and continue from next
		}

		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("heartbeat for %s is not OK, returned: " +resp.Status)
		}
	}
}
