package namenode

import (
	"github.com/gin-gonic/gin"
	"log"
)

var fileToChunkMap map[string][]string // a map to store (str,vector<str>) -> file1 : chunk102, chunk56, chunk 32
// will store the file and the chunks its mapped to

var chunkIDToDatanode map[string][]string // a map which stores all the datanodes which has a partoular chunk
// eg -> chunk1 -> datanode2, datanode4, datanode5

func namenode(){

	r := gin.Default()

	// this endpoint recieves a file
	r.POST("/upload", func(c* gin.Context){
		file, _ := c.FormFile("file")
		log.Println("Namenode recieved file:", file.Filename)
		chunksDataSlice, err := chunkFile(file,file.Filename)
		if err != nil {
			log.Println("Chunking failed: ", err)
		}
		// printing all the contents of the chunksDataSlice array
		for _, chunk := range chunksDataSlice {
			log.Printf("chunkID:%d, index:%d, size:%d ", c.chunkID, c.Index, c.Size)
		}

		// todo: Implement logic to send the chunks to datanodes

	})

