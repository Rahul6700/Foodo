package datanode

import (
	"github.com/gin-gonic/gin"
	"log"
	"models/models"
)

type CompleteChunkData struct {
	chunkID string
	Filename string
	Index int
	Size int
	Data []byte // the actual content
}

func main(){
	r := gin.Default()

	r.POST("/store", func(c* gin.Context){
		var tempStruct models.fileChunk
		err := c.BindJSON(&tempStruct)
		log.Printf("datanode recieved: %s -> %s", tempStruct.MetaData.chunkID, tempStruct.chunkData)
		key := tempStruct.chunkID // key is a string -> chunkID
		var value models.CompleteChunkData // value is a CompleteChunkData which is basically MetaData with the bin content
		value.chunkID = tempStruct.chunkID
		value.Filename = tempStruct.Filename
		value.Index = tempStruct.Index
		value.Size = tempStruct.Size
		value.Data = tempStruct.Data
		store(key,value) // calls the store function in map.go -> stores to the hashmap
	})

	r.POST()
}
