package namenode

import (
	"log"
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/raft"
)

// struct that stores a referene to the Raft node - "raft"
type ApiServer struct {
	raft *raft.Raft
	fsm *FSM
}

// this method creates a new namenode server
// we pass in our main raftNode object
func NewApiServer(r *raft.Raft, fsm *FSM) *ApiServer {
	return &ApiServer{
		raft: r,
		fsm: fsm,
	}
}

// This is where the RAFT server interacts with GIN to expose endpoints
// we have "/status" which tells whether a Raft Namenode is a the leader or not
// "/raft/purpose" endpoints listens to the proposed plan that the LB sends
func (server *ApiServer) RegisterRoutes(r *gin.Engine) {
	r.GET("/status", server.handleStatus)
	r.POST("/raft/propose", server.handlePropose)
	r.GET("/get-metadata", server.handleGetMetadata)
}

// this endpoint is used by the LB to find whether the namenode is the leader or no, return true or false accordingly
func (s* ApiServer) handleStatus (c* gin.Context) {
	if s.raft.State() != raft.Leader {
		c.JSON(503, gin.H{"status" : "false"})
		return
	}
	c.JSON(200, gin.H{"status" : "true"})
}

// this endpoint takes the proporsal from the LB and stores it in the namenode cluster
func (s *ApiServer) handlePropose(c *gin.Context) {
	// fisrt checks if the node selected is the leader
	// only one node (the leader) is allowed to accept data at a time
	// this is done to ensure that 2 namenodes dont accept data simulatinously
	if s.raft.State() != raft.Leader {
		c.JSON(503, gin.H{"error": "not the leader, cannot propose"})
		return
	}

	// read the raw JSON (the RaftCommand) from the LB's request, basically the content of the POST req that the LB sends
	cmdBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "bad request body"})
		return
	}

	//Under the Hood -> This call blocks and triggers a full consensus protocol
	//the leader (this node) writes cmdBytes to its own log file on disk (in the data dir)
	//the leader sends cmdBytes to all other follower namenodes over the private Raft network which is over some port
	//the followers write cmdBytes to their log files and send an ACK to the Leader
	//the leader waits until it has ACKs from a quorum (a majority, like itself + one follower here for our case of 3 NN's)
	//once it has a quorum, the command is Committed -> it is now permanent, even if the leader crashes
	//after its committed raft calls the fsm.Apply() function, which updates the maps
	//only after all of that does this s.raft.Apply() call unblock and returns
	applyFuture := s.raft.Apply(cmdBytes, 5*time.Second) // setting a 5 seconds time out, if the other NN's dont reply within 5 sec's (they are offline), it return false and aborts
	
	// check if the aboe process failed
	if err := applyFuture.Error(); err != nil {
		log.Printf("raft apply error in api.go: %s\n", err)
		c.JSON(503, gin.H{"error": "failed to apply raft command"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *ApiServer) handleGetMetadata(c *gin.Context) {
	// Only the leader should answer read requests
	// to prevent serving "stale" (old) data.
	if s.raft.State() != raft.Leader {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "not the leader"})
		return
	}

	// 1. Get the filename from the query: /get-metadata?filename=foo.txt
	fileName := c.Query("filename")
	if fileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'filename' query parameter"})
		return
	}

	// 2. Call your new thread-safe "getter" on the FSM
	//    (You must pass 'fsm' to your ApiServer, or make this
	//     call in a way that can access the fsm)
    //
    //    !! A common pattern is to pass the FSM into NewApiServer:
    //    type ApiServer struct {
    //        raft *raft.Raft
    //        fsm  *Fsm  // <-- ADD THIS
    //    }
	plan, err := s.fsm.GetFileMetadata(fileName) // <-- ASSUMES 'fsm' IS AVAILABLE
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 3. Send the plan back to the Load Balancer
	c.JSON(http.StatusOK, gin.H{"chunks": plan})
}
