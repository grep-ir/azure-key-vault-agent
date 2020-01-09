package configwatcher

import (
	"context"
	"github.com/chrisjohnson/azure-key-vault-agent/configparser"
	"github.com/chrisjohnson/azure-key-vault-agent/sinkworker"
	"github.com/fsnotify/fsnotify"
	"log"
)

func ConfigWatcher(path string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	//Parse config and start workers.  Get the cancel function back so it can be passed to the file configwatcher
	cancel := parseAndStartWorkers(path)

	//Now that the workers have been started, watch the config file and bounce them if changes happen
	go doWatch(watcher, cancel, path)

	err = watcher.Add(path)
	if err != nil {
		log.Fatal(err)
	}
	<-done //block until done
}

func parseAndStartWorkers(path string) context.CancelFunc {
	// create background context for workers
	ctx, cancel := context.WithCancel(context.Background())
	// if something goes wrong, cancel the set of workers
	defer cancel()

	// parse config file and start workers
	sinkConfigs := configparser.ParseConfig(path)
	for _, sinkConfig := range sinkConfigs {
		go sinkworker.Worker(ctx, sinkConfig)
	}
	return cancel
}

func doWatch(watcher *fsnotify.Watcher, cancel context.CancelFunc, path string) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
				// kill workers and start new ones
				cancel()
				cancel = parseAndStartWorkers(path)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}
