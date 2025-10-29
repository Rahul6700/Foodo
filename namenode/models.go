package namenode

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



