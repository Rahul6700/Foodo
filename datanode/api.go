package datanode

import (
	"log"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"github.gin-gonic/gin"
)

// atomic int a thread safe int (no race conditions as only one thread can modify at a time)
// we use it to count active writes for the DN
var activeWrites atomic.int32

type ApiServer struct {
	dataDir string // Holds the path to the data directory
}

// NewApiServer is the constructor
func NewApiServer(dataDir string) *ApiServer {
	os.MkdirAll(dataDir, 0700) //idempotent data dir creation
	return &ApiServer{dataDir: dataDir}
}

func handleWriteChunk(c* gin.Context, dataDir string)
{
	// as its handling a write rn, we increment the counter
	activeWrites.Add(1)
	// we defer the activeWrites.Sub as it'll ensure the counter is always decremented when the function ends
	defer activeWrites.Add(-1)

	chunkID := c.Param("chunkID") // reads the chunk ID from the URL (query param)
	filePath := filepath.Join(dataDir, chunkID) // we create the file path (/dn-1/chunkID)

	file, err := os.Create(filePath)
	if err != nil {
		c.JSON(500, gin.H{"error" : "couldnt create file in DN for" + chunkID})
		return
	}
	defer file.Close()

	// we use io.Copy(destination, source) to write the content from the req body to the file
	_, err := io.Copy(file, c.Request.Body)
	if err != nil {
		c.JSON(500, gin.h{"error" : "count not write content to file in DN for chunk:" + chunkID})
		return
	}

	log.Printf("successfully wrote chunk %s\n", chunkID)
	c.JSON(200, gin.H{"success" : true})
}

func handleReadChunk(c* gin.Context, dataDir string)
{
	chunkID := c.Param("chunkID") // again extract chunkID from req param
	filePath := filepath.Join(dataDir, chunkID)
	c.File(filePath) // this method finds the file, handles error, setts correct http headers and puts the file from disk into the req body
}
