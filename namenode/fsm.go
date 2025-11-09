package namenode

import (
	"encoding/json"
	"io"
	"log"
	//"github.com/Rahul6700/Foodo/shared"
	"github.com/hashicorp/raft"
	"fmt"
	"sync"
)

// temp struct used only for this file -> restore and snapshot func
type fsm_snapshot struct {
		Files  map[string][]string
		Chunks map[string][]string
	}

type RaftCommand struct {
	Operation string        `json:"operation"`
	Filename  string        `json:"filename"`  
	Chunks    []ChunkStruct `json:"chunks"`
}

type ChunkStruct struct {
	ChunkID    string   `json:"chunk_id"`
	ChunkIndex int      `json:"chunk_index"`
	Locations  []string `json:"locations"`
}

type HeartbeatPayload struct {
	NodeID       string `json:"node_id"`
	ActiveWrites int    `json:"active_writes"`
}

type FSM struct {
	lock                sync.Mutex // Your lock
	fileToChunksMap     map[string][]string
	chunkIDToDataNodesMap map[string][]string
}

type fsmSnapshot struct {
	data []byte
}

// we return a pointer to the FSM, so that whereever its modified from, we always acess the same FMS
// if we pass the FSM directly, copies might get created
// this function creates a new FSM, so everytime our pgm runs a new FSM is created
// even tho a new fsm is created, we dont lose our data on every restart, cuz the raft library handles the syncronisation and populates the map again
func NewFsm() *FSM {
	return &FSM {
			fileToChunksMap: make(map[string][]string),
			chunkIDToDataNodesMap: make(map[string][]string),
	}
}

// to implement : apply, snapshot and restore

// Apply function, 
func (the_fsm *FSM) Apply (raftLog *raft.Log) interface{} {
	the_fsm.lock.Lock()
	defer the_fsm.lock.Unlock()

	 log.Printf("Raw raft log data: %s\n", string(raftLog.Data))

	var cmd RaftCommand // it has cmd.Filename
	log.Printf("0. apply func is applying to cmd.Filename as %s\n", cmd.Filename)
	if err := json.Unmarshal(raftLog.Data, &cmd); err != nil {
		return fmt.Errorf("could not unmarshal command: %s", err)
	}
	log.Printf("1. apply func is applying to cmd.Filename as %s\n", cmd.Filename)
	if cmd.Operation != "REGISTER_FILE"{
		return fmt.Errorf("unknown operation %s", cmd.Operation)
	} else {
		var chunkIDSlice []string
		for _, chunk := range cmd.Chunks {
			chunkIDSlice = append(chunkIDSlice, chunk.ChunkID) // basically all the chunks of the file come in this slice
			the_fsm.chunkIDToDataNodesMap[chunk.ChunkID] = chunk.Locations // this add's data to the fsm's map
			// so what is added is -> fileToChunksMap[chunkID 13434] = [DataNode3, Datanode5, DateNode6]
		}
		log.Printf("2. apply func is applying to cmd.Filename as %s\n", cmd.Filename)
		the_fsm.fileToChunksMap[cmd.Filename] = chunkIDSlice // here we add the file to chunk ID's mapping to the fsm
		// like fileToChunksMap["hello.txt"] = [1312412,3463563463,3453453,23423423] -> id's of the different chunks
	}
	return nil // returning nil if the function runs successfully
}

// the snapshot function takes a snapshot of both the slices and sends it to the FSM
// this function return 2 things, a value and an error
func (the_fsm *FSM) Snapshot() (raft.FSMSnapshot, error) {
	the_fsm.lock.Lock()
	defer the_fsm.lock.Unlock() 

	// 1. Create your "data" struct
	snapshot := fsm_snapshot{
		Files:  the_fsm.fileToChunksMap,
		Chunks: the_fsm.chunkIDToDataNodesMap,
	}

	// 2. Convert it to bytes
	data, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}

	// 3. Return the NEW helper struct, passing the bytes into it
	return &fsmSnapshot{data: data}, nil
}

// --- This code now correctly matches the 'fsmSnapshot' struct ---
func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := sink.Write(s.data); err != nil { 
		return err
	}
	return sink.Close()
}

func (s *fsmSnapshot) Release() {}

// this pulls the stored snapshots from the FSM
// it returns a file, io.ReadCloser technically which is a in built method in "io"
// in traditional io.Read() we can just read, in io.ReadCloser we need to close the fd (it's to save resources)
// if we do io.ReadCloser we need to do defer rc.Close()
func (the_fsm *FSM) Restore (rc io.ReadCloser) error {
	defer rc.Close()
	// a new obj of the fsm_snapshot struct to store the incoming data
	var data fsm_snapshot;

	// creates a NewDecoder obj that reads the message from rc and Decodes it and stores it in the data var
	err := json.NewDecoder(rc).Decode(&data)
	if err != nil {
		return err
	}

	// now we need to write this data to the FSM
	// we lock the FSM first so only this process is accessing it at a time
	the_fsm.lock.Lock()
	defer the_fsm.lock.Unlock()

	the_fsm.fileToChunksMap = data.Files
	the_fsm.chunkIDToDataNodesMap = data.Chunks

	return nil
}

// this is a thread-safe "read-only" function.
// It locks the maps, finds the file's chunks, and builds a plan.
func (f *FSM) GetFileMetadata(fileName string) ([]ChunkStruct, error) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Find the file's chunk IDs
	log.Printf("file name is: %s", fileName)

	log.Printf("Current file map: %v", f.fileToChunksMap)

	chunkIDs, ok := f.fileToChunksMap [fileName]
	if !ok {
		return nil, fmt.Errorf("file %s not found", fileName)
	}

	// build the "plan" by looking up each chunk's location
	var plan []ChunkStruct
	for i, chunkID := range chunkIDs {
		locations, ok := f.chunkIDToDataNodesMap[chunkID]
		if !ok {
			// this means our metadata is corrupt.
			return nil, fmt.Errorf("chunk %s (part of %s) has no location data", chunkID, fileName)
		}
		
		plan = append(plan, ChunkStruct{
			ChunkID:    chunkID,
			ChunkIndex: i,
			Locations:  locations,
		})
	}
	
	return plan, nil
}
