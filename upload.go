package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AnimeNL/joomla-backup/internal/config"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

// Upload file to sftp server
func uploadBackup(localFile, remoteFile string) (err error) {

	// Upload is not done if it's a dryrun (testing)
	if config.Configuration.Global.Dryrun {
		log.Warn("Dryrun. Will not upload.")
		return
	}

	defer conn.Close()
	// Create new SFTP client
	sc, err := sftp.NewClient(conn)
	if err != nil {
		log.Errorf("unable to start SFTP subsystem: %v", err)
	}
	defer sc.Close()

	// Upload file to SFTP
	log.Infof("uploading [%s] to [%s] ...", localFile, remoteFile)

	srcFile, err := os.Open(localFile)
	if err != nil {
		log.Errorf("unable to open local file: %v", err)
		return
	}
	defer srcFile.Close()

	// Make remote directories recursion
	parent := filepath.Dir(remoteFile)
	path := string(filepath.Separator)
	dirs := strings.Split(parent, path)
	for _, dir := range dirs {
		path = filepath.Join(path, dir)
		sc.Mkdir(path)
	}

	// Note: SFTP To Go doesn't support O_RDWR mode
	dstFile, err := sc.OpenFile(remoteFile, (os.O_WRONLY | os.O_CREATE | os.O_TRUNC))
	if err != nil {
		log.Errorf("unable to open remote file: %v\n", err)
		return
	}
	defer dstFile.Close()

	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		log.Errorf("unable to upload local file: %v", err)
		os.Exit(1)
	}
	log.Infof("%d bytes copied", bytes)

	return
}
