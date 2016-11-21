package watch

import (
	"log"
	"os"
	"path"
	"time"
)

func procStable(f string) {
	if _, err := os.Stat(f); err == nil {
		log.Printf("processing %v", f)
		time.Sleep(5 * time.Second) // simulate long running process
		log.Printf("processed %v", f)
	} else {
		log.Printf("%v no longer exists", f)
	}
}
func ExampleNewWatcher() {
	log.Println("running example")
	monPath := "."
	w := NewWatcher(monPath, time.Second, 3*time.Second)
	wt := w.Watch()
	for {
		select {
		case f := <-wt:
			procStable(path.Join(monPath, f))
		}
	}
}
