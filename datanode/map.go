package datanode

import (
	"models/models"
	"log"
)

var mp map[string]models.CompleteChunkStruct // mp[chunkID] = bin byte for that chunk

func store(chunkID string, data models.CompleteChunkStruct){
	mp[chunkID] = data;
	log.Println("stored %s from %s in datanode", chunkID, data.Filename)
}

