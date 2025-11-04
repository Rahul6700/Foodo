package shared

// we'll import this package and use it in our, loadb
// this acts as the common storage struct for communication between them


// this is the truct for the RaftCommmand
type RaftCommand struct {
	Operation string
	Filename string
	Chunks []ChunkStruct
}

// this is the helper struct
type ChunkStruct struct {
	ChunkID string
	ChunkIndex int
	Locations []string
} 
