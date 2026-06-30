package main

import (
	"fmt"

	"github.com/google/uuid"
)

func main() {
	id, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", id.String())
	fmt.Printf("%#v\n", id)
}
