package main

import (
	"flag"
	"log"
	"os"
	"github.com/Rahul6700/Foodo/datanode" 
	"github.com/gin-gonic/gin"
)

var (
	// the public facing port
	apiAddr = flag.String("api-addr", ":9001", "API address (e.g., :9001)")
	//the folder to store chunks
	dataDir = flag.String("data-dir", "dn-data-1", "Data directory for chunks")
	//The addr (url) of the loadb
	lbAddr = flag.String("lb-addr", "", "Load Balancer address, sumn like -> http://192.168.1.10:8000)")
)

func main() {
	flag.Parse()
	if *lbAddr == "" {
		log.Fatal("Load Balancer address is required")
	}
	// idempotent dir creation to store chunks
	os.MkdirAll(*dataDir, 0700)

	// we start the hearBeat sending process in the BG using a goroutine
	// we give it the LB addr so it can send there and the public port on which it can recieve responeses
	go datanode.StartHeartBeat(*lbAddr, *apiAddr)

	api := datanode.NewApiServer(*dataDir)
	
	r := gin.Default()
	// Pass the dataDir to the route handlers so they know where to save files
	r.POST("/writeChunk/:chunkID", api.HandleWriteChunk)
	r.GET("/readChunk/:chunkID", api.HandleReadChunk)

	log.Printf("Datanode API server starting on %s\n", *apiAddr)
	// We listen on 0.0.0.0 to be reachable from other machines
	if err := r.Run("0.0.0.0" + *apiAddr); err != nil {
		log.Fatalf("Datanode API server failed: %s", err)
	}
}
