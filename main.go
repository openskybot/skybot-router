package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal(fmt.Sprintf("Usage: %s uavobject_directory", os.Args[0]))
	}

	loadUAVObjectDefinitions(os.Args[1])

	c := startUAVTalk()

	for uavTalkObject := range c {
		uavdef := getUAVObjectDefinitionForObjectID(uavTalkObject.objectId)

		if uavdef != nil {
			fmt.Println(uavTalkObject)
			fmt.Println(uavdef)
		} else {
			fmt.Printf("!!!!!!!!!!!! Not found : %d !!!!!!!!!!!!!!!!!\n", uavTalkObject.objectId)
		}
		fmt.Println("")
	}

	return
}