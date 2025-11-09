package shared

// we'll import this package and use it in our, loadb
// this acts as the common storage struct for communication between them


// this is the truct for the RaftCommmand
type RaftCommand struct {
	Operation string `json:"operation"`
	Filename string `json:"filename"`
	Chunks []ChunkStruct `json:"chunks"`
}

// this is the helper struct
type ChunkStruct struct {
	ChunkID string `json:"chunk_id"`
	ChunkIndex int `json:"chunk_index"`
	Locations []string `json:"locations"`
}

// HeartbeatPayload is used by the DN's to send heartbeat's to the LB
type HeartbeatPayload struct {
	NodeID       string `json:"node_id"`       // DN's full url -> "http://192.168.1.15:9001"
	ActiveWrites int    `json:"active_writes"` // load tracked by the atomic counter
}
