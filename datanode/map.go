package datanode

import (
	"models/models"
	"log"
)

var mp map[string]models.CompleteChunkStruct // mp[chunkID] = bin byte for that chunk

func store(chunkID string, data models.CompleteChunkStruct){
	mp[chunkID] = data;
	log.Printf("stored %s from %s in datanode", chunkID, data.Filename)
}

func get(chunkID string) {
	return mp[chunkID]
}


