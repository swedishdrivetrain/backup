package main

//TODO: Complete log statements
import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/AnimeNL/joomla-backup/internal/config"
	"github.com/docker/docker/api/types"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
)

var (
	workdir   = "/tmp/jbackup/"
	layoutISO = "2006-01-02"
	dbdump    = workdir + "db/"
	fsdump    = workdir + "fs/"
	dc        = config.Configuration.DockerClient
)

func setup() {
	log.Debugf("creating workdir %v\n", workdir)
	os.Mkdir(workdir, 0755)
	log.Debugf("creating database dump folder %v\n", dbdump)
	os.Mkdir(dbdump, 0755)
	log.Debugf("creating filesystem dump folder %v\n", fsdump)
	os.Mkdir(fsdump, 0755)
}

func cleanup() {
	log.Infoln("cleanup")
	log.Debug("cleaning up db dumps")
	dir, _ := os.Open(config.Configuration.Paths.DatabaseDumps)
	defer dir.Close()
	files, _ := dir.Readdir(0)
	for _, file := range files {
		os.Remove(config.Configuration.Paths.DatabaseDumps + "/" + file.Name())
	}
	log.Debug("remove workdir")
	os.RemoveAll(workdir)
}

func consolidateDatabaseDumps() {
	dir, err := os.Open(config.Configuration.Paths.DatabaseDumps)
	if err != nil {
		log.Errorf("error opening dir %v\n", err)
	}
	files, err := dir.Readdir(0)
	if err != nil {
		log.Errorf("error reading dir %v\n", err)
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			err := os.Rename(config.Configuration.Paths.DatabaseDumps+"/"+file.Name(), dbdump+file.Name())
			if err != nil {
				log.Errorf("unable to move file: %v\n", err)
			}
		}
	}
	dir.Close()
}

func databaseDump(ctx context.Context, database string) {
	command := []string{"bash", "-c", "/usr/bin/mysqldump -u " + config.Configuration.Database.Credentials.Username + " --password=" + config.Configuration.Database.Credentials.Password + " " + database + " > /dump/" + database + ".sql"}
	log.Debugf("constructed docker exec command: %v\n", command)
	execConfig := types.ExecConfig{Tty: false, AttachStdout: true, AttachStderr: false, Cmd: command}
	respIdExecCreate, err := dc.ContainerExecCreate(ctx, "mysql", execConfig)
	if err != nil {
		fmt.Println(err)
	}
	_, err = dc.ContainerExecAttach(ctx, respIdExecCreate.ID, types.ExecStartCheck{})
	if err != nil {
		fmt.Println(err)
	}
}

func fileDump(srcPath string, destFile string) error {
	// compress file or folder
	var buf bytes.Buffer
	if err := compress(srcPath, &buf); err != nil {
		log.Errorf("Error compressing directory %v: %v\n", srcPath, err)
		return err
	}
	if err := ioutil.WriteFile(destFile+".tar.gz", buf.Bytes(), 0660); err != nil {
		log.Error("Unable to write compressed file: %v\n", err)
		return err
	}
	return nil
}

func main() {
	setup()
	defer cleanup()
	ctx := context.Background()
	conn := config.Configuration.SSHClient

	// Dump all databases
	for _, database := range config.Configuration.Database.Databases {
		log.Infof("Dumping database %v\n", database)
		databaseDump(ctx, database) //TODO: Use goroutines to dump databases in parallel and make the backup more efficent
	}
	// Gather dumps in workdir
	consolidateDatabaseDumps()

	// Compress all data directories
	for _, path := range config.Configuration.Paths.FileDumps {
		log.Infof("Compressing %v\n", path)
		fileDump(path, fsdump+filepath.Base(path)) //TODO: Use goroutines to compress filesystems in parallel and make the backup more efficent
	}

	// Compress full backup
	time := time.Now()
	date := time.Format(layoutISO)

	log.Info("Compressing backup")
	fileDump(workdir, "/tmp/backup-"+date)

	defer conn.Close()

	// Create new SFTP client
	sc, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatalf("Unable to start SFTP subsystem: %v\n", err)
	}
	defer sc.Close()

	// Upload to SFTP site
	log.Info("Uploading backup to store")
	uploadBackup(sc, "/tmp/backup-"+date+".tar.gz", "backup-"+date+".tar.gz")
	// Cleanup
	os.Remove("/tmp/backup-" + date + ".tar.gz")
	log.Info("exiting")
}
