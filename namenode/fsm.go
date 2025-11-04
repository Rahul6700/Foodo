package namenode

import (
	"encoding/json"
	"bytes"
	"io"
	"foodo/shared"
	"food/model"
	"log"
)

// we return a pointer to the FSM, so that whereever its modified from, we always acess the same FMS
// if we pass the FSM directly, copies might get created
// this function creates a new FSM, so everytime our pgm runs a new FSM is created
// even tho a new fsm is created, we dont lose our data on every restart, cuz the raft library handles the syncronisation and populates the map again
func NewFsm() *FSM {
	return &FSM {
			fileToChunksMap: make(map[string][string]),
			chunkIDToDataNodesMap: make(map[string][]string)
	}
}

// to implement : apply, snapshot and restore

// Apply function, 
func (the_fsm *FSM) Apply (log *raft.log) interface{} {
	the_fsm.lock.Lock()
	defer the_fsm.lock.Unlock()

	var cmd shared.RaftCommand
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return log.Printf("could not unmarshal command: %s\n", err)
	}

	if(cmd.Operation != "REGISTER_FILE") return log.Printf("unknown operation %s\n", cmd.Operation)
	else {
		var chunkIDSlice []string
		for _, chunk := range cmd.Chunks {
		}
	}
}
