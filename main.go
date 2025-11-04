package main

import (
	"fmt"
)

func main(){
	fmt.Println("HelloWorld")
	s := []int{}
	for i := 0; i < 10; i++ {
		s = append(s,i);
	}

	for index ,num := range s {
		fmt.Printf("index %d has %d\n", index,num)
	}
}
