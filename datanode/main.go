package datanode

import (
	"github.com/gin-gonic/gin"
	"log"
	"models/models"
)

func main(){
	r := gin.Default()

	r.POST("/store", func(c* gin.Context){
		var tempStruct models.fileChunk
		err := c.BindJSON(&tempStruct)
		log.Println("datanode recieved: %s -> %s", tempStruct.MetaData.chunkID, tempStruct.chunkData)
	})
}
