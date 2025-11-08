package main

import (
	"fmt"
	"flag"
	"log"
	"net"
	"os"
	"time"
	"foodo/namenode"
	"path/filepath"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb" 
)


// when we run our main func, we add flags with it like -> go run ./namenode/ -id=nn-1 -api-addr=:8001 -raft-addr=:7001 -data-dir=./data-1 -bootstrap
// so we extract all these flags from the go run command and use them elsewhere (to create our raft node)
// this is just a reusable var to store those flags
var (
	nodeID    = flag.String("id", "", "Node ID") // unique ID for the node
	apiAddr   = flag.String("api-addr", ":8001", "API address") // the public IP add for the Gin API endpoint (8001 is the fallback value incase users dont specify in the go run cmd)
	raftAddr  = flag.String("raft-addr", ":7001", "Raft address") // a pvt address for raft nodes to talk to each other
	dataDir   = flag.String("data-dir", "data-1", "Data directory") // the dir where we store the namenodes's data (given )
	bootstrap = flag.Bool("bootstrap", false, "Bootstrap cluster") // bootstrap flag with value as true or false, true if this is the first node to start (automatically becomes leader without election)
)

func main(){
	// first read all the flags from the run cmd
	flag.Parse()
	if *nodeID == "" {
		log.Println("nodeID not set")
		return
	}

	// setup
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(*nodeID)
	os.MkdirAll(*dataDir, 0700) // this is idempotent data dir creation (is dir doesnt exist, create one. else use existing one). 0700 is rwx permission for the user

	// the hashicorp/raft library we are using uses boltStore which is a key-val pair built on top of BoltDB (an embedded DB) to store our content
	//then we create the 3 needed files iniside the dir we just made
	//log.dat is the log file that hold records of all the raft logs made. This is what feeds the FSM.
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(*dataDir, "logs.dat"))
	if err != nil {
		log.Printf("error creating logs.dat: %s", err)
	}
	// in raft events happen in terms of "terms", keeps track of what has happened in the latest term and who is voted for.
	// this is so that in case the server restarts, it can see this file and know whats happening instead of corrupting the entire system.
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(*dataDir, "stable.dat"))
	if err != nil {
		log.Printf("error creating stable.dat: %s", err)
	}
	// this is where the snapshots are saved for the node that can be restored later
	snapshotStore, err := raft.NewFileSnapshotStore(*dataDir, 2, os.Stdout)
	if err != nil {
		log.Printf("error creating snapshot store: %s", err)
	}

	// createa pvt network for raft commuinication to happen between the namenodes
	addr, err := net.ResolveTCPAddr("tcp", *raftAddr) // a correct raft address is formulated for the TCP connection
	if err != nil {
		log.Printf("error creating addr for the pvt NN tcp: %s", err)
	}
	transport, err := raft.NewTCPTransport(*raftAddr, addr, 3, 10*time.Second, os.Stdout) // 3 is the number of persistent connections each node maintains with other nodes. so node 1 will have 3 per connections with node 2 and 3 with node 3.
	if err != nil {
		log.Printf("error creating pvt TCP conn for NN's : %s", err)
	}

	//create a new fsm
	fsm := namenode.NewFsm()

	// building a raft node object, we take all the params we created up and use them here to construct the raft node
	// if the values we pass are not proper, the obj wont be created and we get an error message, so we'll have an err handling for this down
	// this function too is idempotent, it first checks snapshots and if something is found, restores
	raftNode, err := raft.NewRaft (
		config,
		fsm,
		logStore,
		stableStore,
		snapshotStore,
		transport
	)
	if err != nil { // obj not created
		log.Printf("raftNode obj not created: %s", err)
	}

	// we check if the bootsrap flag has been provided
	// if it has. then it means this is the first node created and hence it directly becomes the leader node
	// we give it a "guest-list" basically of all the nodes which can be allowed as namenodes
	// the bootstrapped node write this list to its logs.dat and becomes leader
	// it does not call or try communicating to the other nodes, so even if they are not active, its fine. it just stores the list
	if *bootstrap {
		log.Printf("Bootstrapping using NN:%d", nodeID)
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{ ID: "nn-1", Address: raft.ServerAddress("localhost:7001") },
				{ ID: "nn-2", Address: raft.ServerAddress("localhost:7002") },
				{ ID: "nn-3", Address: raft.ServerAddress("localhost:7003") },
			},
		}
		f := raftNode.BootstrapCluster(cfg)
	}

	// set up a gin endpoint for the cluster to listen publically
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// "inject" the Raft engine into the API server
	apiServer := namenode.NewApiServer(raftNode) // NewApiServer is the method in api.go that creates a new an api server and passes the raft engine by referrence to it, now the api server has a ptr to the raft engine that it can use to serve
	apiServer.RegisterRoutes(r) // we now pass the router too

	log.Printf("API server starting on %s\n", *apiAddr)
	
	// Starts the server and blocks infinitely
	if err := r.Run(*apiAddr); err != nil {
		log.Fatalf("API server failed: %s", err)
	}
}
