package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func cleanup() {
	log.Infof("cleanup")

	removeBackupFile()
	// go cleanDumps()
	cleanWorkdir()
}

func removeBackupFile() {
	log.Info("cleaning backup file")

	err := os.Remove("/tmp/backup-" + timestamp + ".tar.gz")
	if err != nil {
		log.Errorf("error removing backup file %v", err)
	}
}

// func cleanDumps() {
// 	log.Debug("cleaning up db dumps")
// 	dir, err := os.Open(config.Configuration.Paths.DatabaseDumps)
// 	if err != nil {
// 		log.Errorf("error opening dir %v", err)
// 	}

// 	defer dir.Close()

// 	files, _ := dir.Readdir(0)
// 	for _, file := range files {
// 		err = os.Remove(config.Configuration.Paths.DatabaseDumps + "/" + file.Name())
// 		if err != nil {
// 			log.Errorf("error removing file %v", err)
// 		}
// 	}
// }

func cleanWorkdir() {
	log.Debug("remove workdir")

	if err := os.RemoveAll(workdir); err != nil {
		log.Errorf("error removing workdir: %v", err)
	}

}
