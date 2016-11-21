package watch

import (
	"log"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// NumStableFiles set the size of the stableFiles buffered channel.
const NumStableFiles = 50

// Watcher monitors a given monPath, checks for file updates every
// pollDur and declares them stable if no updates are detected after stableDur.
type Watcher struct {
	monPath            string
	pollDur, stableDur time.Duration
	files              map[string]time.Time
}

// NewWatcher watches a folder for filesystem events.
func NewWatcher(monPath string, pollDur time.Duration, stableDur time.Duration) *Watcher {
	w := Watcher{}
	w.monPath = monPath
	w.pollDur = pollDur
	w.stableDur = stableDur
	w.files = make(map[string]time.Time)
	return &w
}

// Watch returns stable filenames on the channel.
func (w *Watcher) Watch() <-chan string {
	stableFilename := make(chan string, NumStableFiles)
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}
		defer watcher.Close()

		done := make(chan bool)
		evch := make(chan fsnotify.Event)
		monch := w.mon(evch)
		go func() {
			for {
				select {
				case event := <-watcher.Events:
					log.Println("event:", event)
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("modified file:", event.Name)
					}
					evch <- event
				case s := <-monch:
					log.Printf("stable: %v\n", s)
					stableFilename <- s
				case err = <-watcher.Errors:
					log.Println("error:", err)
				}
			}
		}()

		err = watcher.Add(w.monPath)
		if err != nil {
			log.Fatal(err)
		}
		<-done
	}()
	return stableFilename
}

// Watcher:  local un-exported methods
func (w *Watcher) timeStampFiles() {
	now := time.Now()
	g, err := filepath.Glob(filepath.Join(w.monPath, "*"))
	if err != nil {
		log.Fatal("filepath:", err)
	}
	for _, v := range g {
		w.files[v] = now
	}
}
func (w *Watcher) mon(ev <-chan fsnotify.Event) <-chan string {
	w.timeStampFiles()

	ch := make(chan string)
	poll := time.Tick(w.pollDur)
	go func(e <-chan fsnotify.Event, s chan<- string) {

		var now time.Time
		for {
			select {
			case event := <-e:
				eventName := filepath.Clean(event.Name) // eventName is the cleaned absolute pathname of the file associated with the event
				switch {
				case event.Op&fsnotify.Create == fsnotify.Create:
					w.files[eventName] = time.Now()
				case event.Op&fsnotify.Write == fsnotify.Write:
					w.files[eventName] = time.Now()
				case event.Op&fsnotify.Chmod == fsnotify.Chmod:
					w.files[eventName] = time.Now()
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					delete(w.files, eventName)
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					delete(w.files, eventName)
				}
			case <-poll:
				// check files for stablility and push to s
				now = time.Now()
				for fn, ts := range w.files {
					if now.After(ts.Add(w.stableDur)) {
						delete(w.files, fn)
						s <- fn
					}
				}

			}
		}
	}(ev, ch)
	return ch
}
