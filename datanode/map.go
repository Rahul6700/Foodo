package datanode

import (
		
)

var mp map[string][]byte // mp[chunkID] = bin byte for that chunk

func store(chunkID string, data byte[]){
	mp[chunkID] = data;
}

