package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {

	flag.UintVar(&flagMaxNum, "N", 4, "max amount of processes to work at the same time")
	flag.Parse()

	log.Println(flagMaxNum)
	bufChan = make(chan struct{}, flagMaxNum)

	http.HandleFunc("/", StartPage)
	http.HandleFunc("/add", AddWorker)
	http.HandleFunc("/show", GetWorkerList)

	fmt.Println("listening server on :8080...")
	http.ListenAndServe(":8080", nil)
}
