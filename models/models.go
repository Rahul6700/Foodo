package models

// NameNode Structs -------------

// strcut to store metadata for every chunk
type ChunkMetaData struct {
	chunkID string // each chunk will have a unique ID
	Filename string
	Index int // order of the chunks to reconstruct the origin file
	Size int // size of the chunk
}

// this struct is for the file chunks themselves
type fileChunk struct {
	MetaData ChunkMetaData // the metadata of the chunk
	chunkData []byte // the actual bin chunk
}

// DataNode Structs -------------

// this struct is the value for the key value pair that is stored in datanodes
type CompleteChunkData struct {
	chunkID string
	Filename string
	Index int
	Size int
	Data []byte // the actual content
}

// the struct for the FSM 
type FSM struct {
	lock sync.Mutex // we lock the maps
	fileToChunksMap map[string][]string 
	chunkIDToDataNodesMap map[string][]string
}


