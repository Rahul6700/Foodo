package namenode

import (
	"encoding/json"
	"bytes"
	"io"
	"foodo/shared"
	"food/model"
	"log"
)

// temp struct used only for this file -> restore and snapshot func
type fms_snapshot struct {
		Files  map[string][]string
		Chunks map[string][]string
	}

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
			chunkIDSlice = append(chunkIDSlice, chunk.ChunkID) // basically all the chunks of the file come in this slice
			the_fsm.chunkIDToDataNodesMap[chunk.ChunkID] = chunk.Locations // this add's data to the fsm's map
			// so what is added is -> fileToChunksMap[chunkID 13434] = [DataNode3, Datanode5, DateNode6]
		}
		the_fsm.fileToChunksMap[cmd.Filename] = chunkIDSlice // here we add the file to chunk ID's mapping to the fsm
		// like fileToChunksMap["hello.txt"] = [1312412,3463563463,3453453,23423423] -> id's of the different chunks
	}
	return nil // returning true if the function runs successfully
}

// the snapshot function takes a snapshot of both the slices and sends it to the FSM
// this function return 2 things, a value and an error
func (the_fsm *FSM) Snapshot (raft.FSMSnapshot, error) interface{} {
	the_fsm.lock.Lock()
	defer the_fsm.lock.Unlock() // lock the fsm in the start of the function and unlock after the operation is complete, so only one process can acces
	// it and hence avoiding race conditions

	// we create a snapshot of that struct we just created,
	// so we basically put both the maps (file to chunks and chunksID to DN) in the struct and send a snapshot of it to the FSM
	snapshot := fms_snapshot {
		Files : the_fsm.fileToChunksMap,
		Chunks : the_fsm.chunkIDToDataNodesMap
	}
	// obv, we cant send a go struct, so we convert the snapshot to a binary slice
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err // returning a value and an error
	}
	// on success
	return &fsmSnapshot{data: data}, nil
}

// this pulls the stored snapshots from the FSM
// it returns a file, io.ReadCloser technically which is a in built method in "io"
// in traditional io.Read() we can just read, in io.ReadCloser we need to close the fd (it's to save resources)
// if we do io.ReadCloser we need to do defer rc.Close()
func (the_fsm *FSM) restore (rc io.ReadCloser) error {
	defer rc.Close()
	// a new obj of the fsm_snapshot struct to store the incoming data
	var data fsm_snapshot;

	// creates a NewDecoder obj that reads the message from rc and Decodes it and stores it in the data var
	err := json.NewDecoder(rc).Decode(&data)
	if err != nil {
		return nil
	}

	// now we need to write this data to the FSM
	// we lock the FSM first so only this process is accessing it at a time
	the_fsm.lock.Lock()
	defer the_fsm.lock.Unlock()

	the_fsm.fileToChunksMap = data.Files
	the_fsm.chunkIDToDataNodesMap = data.Chunks

	return nil
}
