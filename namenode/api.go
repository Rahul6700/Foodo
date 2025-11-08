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
}

// this method creates a new namenode server
// we pass in our main raftNode object
func NewApiServer(r *raft.Raft) *ApiServer {
	return &ApiServer{raft: r}
}

// This is where the RAFT server interacts with GIN to expose endpoints
// we have "/status" which tells whether a Raft Namenode is a the leader or not
// "/raft/purpose" endpoints listens to the proposed plan that the LB sends
func (server *ApiServer) RegisterRoutes(r *gin.Engine) {
	r.GET("/status", server.handleStatus)
	r.POST("/raft/propose", server.handlePropose)
}

// this endpoint is used by the LB to find whether the namenode is the leader or no, return true or false accordingly
func (s* ApiServer) handleStatus (c* gin.Context) {
	if server.raft.State() != raft.Leader {
		c.JSON(503, gin.H{"status" : "false"})
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
		c.JSON(400, gin.H{"error": "bad request body"})r
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


