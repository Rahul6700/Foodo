package namenode

// strcut to store metadata for every chunk
type ChunkData struct {
	chunkID string // each chunk will have a unique ID
	Filename string
	Index int // order of the chunks to reconstruct the origin file
	Size int // size of the chunk
}
