package namenode

import (
	"log"
	"crypto/sha1"
)

// 2mb chunkSize
const chunkSize := 2*1024*1024

// this helper functions takes in a the buffer's content (file content chunk) -> so format is a slice of bytes
// and returns a string -> the unique hash ID generated using SHA1
func sha1sum(inp []byte) string {
    h := sha1.New() // hasher obj
    h.Write(inp)
    return hex.EncodeToString(h.Sum(nil)) // EncodeToString() converts the SHA1 output to a hexadecimal string
}

func chunkFile(file io.Reader, filename string) ([]ChunkMetaData, [][]byte, error) {
    var chunksDataSlice []ChunkMetaData // this array is what we will be returning, this is an array of metdata (chunkData) of all the chunks of the file
    var fileChunksSlice [][]byte // this slices contains the bin slices -> each chunk's binary is a []byte, so we return a slice of those ([][]byte)
    buffer := make([]byte, ChunkSize) // we make a buffer of size ChunkSize (2mb)
    index := 0
    for {
        n, err := file.Read(buffer)
        if n > 0 {
            chunkData := make([]byte, n) // creates a new slice of size 'n'
            copy(chunkData, buffer[:n]) // copy n bytes from the buffer into the slice -> this slice is now our chunk bin
            chunkID := hashHelper(chunkData) // get the unique hash Id
            // append the chunks metadata to the chunks array
            chunksDataSlice = append(chunksMetaData, ChunkInfo{chunkID:chunkID, FileName:filename, Index:index, Size:int64(n)}) // writing to metadata slice
            fileChunksSlice = append(fileChunksSlice, chunkData) // writing to chunk bin slice
            index++
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, nil, err
        }
    }
    return chunksDataSlice, fileChunksSlice, nil
}

