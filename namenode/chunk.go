package namenode

import (
	"log"
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

func chunkFile(file io.Reader, filename string) ([]ChunkInfo, error) {
    var chunks []ChunkInfo // this array is what we will be returning, this is an array of metdata (chunkData) of all the chunks of the file
    buffer := make([]byte, ChunkSize)
    index := 0
    for {
        n, err := file.Read(buffer)
        if n > 0 {
            //we want to give each chunk a unique ID, so we compute a unique hash for every chunk using SHA1
            chunkID := hashHelper(buffer[:n]) // get the unique hash ID
            // append the chunks metadata to the chunks array
            chunks = append(chunks, ChunkInfo{chunkID:chunkID, FileName:filename, Index:index, Size:int64(n)})
            index++
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, err
        }
    }
    return chunks, nil
}
