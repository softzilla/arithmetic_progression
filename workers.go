package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Worker struct {
	NumInQueue   int
	Status       string
	ElemQ        int
	ElemToChange float64
	Delta        float64
	Interval     float64 // interval between operations, secs
	TTL          float64 // timeout to save result, secs
	Iteration    uint
	ScheduleTime time.Time
	StartTime    time.Time
	FinishTime   time.Time
}

// constants for Worker::Status field
const Wait = "WAITING"
const Work = "PROCESSING"
const Done = "DONE"

var addFormTmpl = []byte(`
<html>
	<body>
	<form action="/add" method="post">
		ElemQ: <input type="text" name="elemQ"> 
		Delta: <input type="text" name="delta"> 
		ElemFirst: <input type="text" name="elemfirst"> 
		Interval: <input type="text" name="interval"> 
		TTL: <input type="text" name="ttl"> 
		<input type="submit" value="Add worker">
	</form>
	</body>
</html>
`)

// workerPool contains all the processes TTL if which isnt done
var workerPool []*Worker

// flagMaxNum sets max amount of processes is safe to perform
var flagMaxNum uint
var bufChan chan struct{}

// StartPage gives access to form which you can type parameters into
func StartPage(w http.ResponseWriter, r *http.Request) {
	w.Write(addFormTmpl)
}

// AddWorker parses the query and apopends new worker to the workerPool slice
func AddWorker(w http.ResponseWriter, r *http.Request) {

	var worker Worker

	worker.ElemQ, _ = strconv.Atoi(r.FormValue("elemQ"))
	worker.Delta, _ = strconv.ParseFloat(r.FormValue("delta"), 64)
	worker.ElemToChange, _ = strconv.ParseFloat(r.FormValue("elemfirst"), 64)
	worker.Interval, _ = strconv.ParseFloat(r.FormValue("interval"), 64)
	worker.TTL, _ = strconv.ParseFloat(r.FormValue("ttl"), 64)

	worker.ScheduleTime = time.Now()
	worker.Status = Wait

	workerPool = append(workerPool, &worker)

	http.Redirect(w, r, "/", http.StatusFound)

	var queueCounter int // updating the num query
	for _, worker := range workerPool {
		if worker.Status == Wait {
			worker.NumInQueue = queueCounter
			queueCounter++

		}
	}

	go DoWorkerPool()

}

// GetWorkerList sorts workerPool according to their statius firstly, to NumInQueue then; prints all the results out via standart output
func GetWorkerList(w http.ResponseWriter, r *http.Request) {

	sort.Slice(workerPool, func(i, j int) bool {
		if workerPool[i].Status == workerPool[j].Status {
			return workerPool[i].NumInQueue > workerPool[j].NumInQueue
		}
		st := map[string]int{
			Wait: 0,
			Work: 1,
			Done: 2,
		}
		return st[workerPool[i].Status] < st[workerPool[j].Status]
	})

	for _, worker := range workerPool { // parse our dates/times so that the result isn't too long
		dateSchedule := parsedate(worker.ScheduleTime)
		if dateSchedule == "0:0:0" {
			dateSchedule = ""
		}
		dateStart := parsedate(worker.StartTime)
		if dateSchedule == "0:0:0" {
			dateStart = ""
		}
		dateFinish := parsedate(worker.FinishTime)
		if dateSchedule == "0:0:0" {
			dateFinish = ""
		}

		w.Write([]byte(fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%vs\t%v\t%v\t%v\t%v\t%v\t\n",
			worker.NumInQueue, worker.Status, worker.ElemQ, worker.ElemToChange,
			worker.Delta, worker.Interval, worker.Iteration, dateSchedule, dateStart, dateFinish, worker.TTL)))
	}
}

func parsedate(t time.Time) string {
	h, min, s := t.Clock()
	// y, m, d := t.Date() // we can leave the date if needed
	return fmt.Sprintf("%v:%v:%v", h, min, s)
}

// DoWorkerPool launches goroutins to process if bufChan is available
func DoWorkerPool() {

	for _, worker := range workerPool {
		if worker.Status == Wait && worker.NumInQueue == 0 {
			bufChan <- struct{}{}
			go DoOneWorker(worker)
		}
	}
}

// DoOneWorker n
func DoOneWorker(worker *Worker) {

	worker.StartTime = time.Now()
	worker.Status = Work

	for i := 0; i < worker.ElemQ; i++ {
		time.Sleep(time.Millisecond * time.Duration(worker.Interval*1000)) // округлили до тысячных.. можно до большей точности вплоть до 10ˆ9
		worker.ElemToChange += worker.Delta
		worker.Iteration++
	}

	worker.Iteration = 0
	worker.FinishTime = time.Now()
	worker.Status = Done

	<-bufChan

	time.AfterFunc(time.Millisecond*time.Duration(worker.TTL*1000), func() {
		deleteElem(worker)
	})
}

func deleteElem(w *Worker) {
	mu := sync.Mutex{}

	for indx, item := range workerPool {
		if item == w {
			mu.Lock()
			workerPool[indx], workerPool[len(workerPool)-1] = workerPool[len(workerPool)-1], workerPool[indx]
			workerPool = workerPool[:len(workerPool)-1]
			mu.Unlock()

			return
		}
	}
}
