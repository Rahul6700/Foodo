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

	})
}
