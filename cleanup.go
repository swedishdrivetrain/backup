package main

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func cleanup() {
	log.Infof("cleanup")

	removeBackupFile()
	cleanWorkdir()
}

func removeBackupFile() {
	log.Info("cleaning backup file")

	err := os.Remove("/tmp/backup-" + timestamp + ".tar.gz")
	if err != nil {
		log.Errorf("error removing backup file %v", err)
	}
}

func cleanWorkdir() {
	log.Debug("remove workdir")

	if err := os.RemoveAll(workdir); err != nil {
		log.Errorf("error removing workdir: %v", err)
	}

}
