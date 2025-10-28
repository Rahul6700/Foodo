package namenode

import (
	"github.com/gin-gonic/gin"
	"log"
)

var mp map[string][]string // a map to store (str,vector<str>)
// will store the file and the datanodes its mapped to

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
			log.Println("chunkID:%d, index:%d, size:%d ", c.chunkID, c.Index, c.Size)
		}

		// todo: Implement logic to send the chunks to datanodes

	})
}
