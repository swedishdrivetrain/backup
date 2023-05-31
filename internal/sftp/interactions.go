package sftp

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	c "joomla-backup/internal/config"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

func ListBackups() ([]fs.FileInfo, error) {
	conn, err := initSftp()
	if err != nil {
		log.Fatalf("Error opening SSH connection: %v", err.Error())
	}
	defer conn.Close()

	// Create new SFTP client
	sc, err := sftp.NewClient(conn)
	if err != nil {
		log.Errorf("unable to start SFTP subsystem: %v", err)
	}
	defer sc.Close()

	files, err := sc.ReadDir("/")
	if err != nil {
		log.Errorf("unable to list files in SFTP remote: %v", err)
	}

	return files, nil
}

// Upload file to sftp server
func UploadBackup(localFile, remoteFile string) (err error) {
	conn, err := initSftp()
	if err != nil {
		log.Fatalf("Error opening SSH connection: %v", err.Error())
	}

	// Upload is not done if it's a dryrun (testing)
	if c.Configuration.Global.Dryrun {
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
		log.Fatalf("unable to upload local file: %v", err)
	}
	log.Infof("%d bytes copied", bytes)

	return
}

func DeleteBackup(remoteFile string) (err error) {
	conn, err := initSftp()
	if err != nil {
		log.Fatalf("Error opening SSH connection: %v", err.Error())
	}
	defer conn.Close()

	// Create new SFTP client
	sc, err := sftp.NewClient(conn)
	if err != nil {
		log.Errorf("unable to start SFTP subsystem: %v", err)
	}
	defer sc.Close()

	err = sc.Remove(remoteFile)
	if err != nil {
		return err
	}

	return nil
}
