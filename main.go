package main

//TODO: Complete log statements
import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/AnimeNL/joomla-backup/internal/config"
	"github.com/docker/docker/api/types"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	workdir   = "/tmp/jbackup/"
	layoutISO = "2006-01-02T15:04:05"
	dbdump    = workdir + "db/"
	fsdump    = workdir + "fs/"
	dc        = config.Configuration.DockerClient
	timestamp string
	ctx       context.Context
	conn      *ssh.Client
)

func setup() {
	// set timestamp of backup
	time := time.Now()
	timestamp = time.Format(layoutISO)
	ctx = context.Background()
	conn = config.Configuration.SSHClient

	log.Debugf("creating workdir %v", workdir)
	os.Mkdir(workdir, 0755)
	log.Debugf("creating database dump folder %v", dbdump)
	os.Mkdir(dbdump, 0755)
	log.Debugf("creating filesystem dump folder %v", fsdump)
	os.Mkdir(fsdump, 0755)
}

func cleanup() {
	log.Infof("cleanup")

	go removeBackupFile()
	go cleanDumps()
	go cleanWorkdir()
}

func removeBackupFile() {
	log.Info("cleaning backup file")

	err := os.Remove("/tmp/backup-" + timestamp + ".tar.gz")
	if err != nil {
		log.Errorf("error removing backup file %v", err)
	}
}

func cleanDumps() {
	log.Debug("cleaning up db dumps")
	dir, err := os.Open(config.Configuration.Paths.DatabaseDumps)
	if err != nil {
		log.Errorf("error opening dir %v", err)
	}

	defer dir.Close()

	files, _ := dir.Readdir(0)
	for _, file := range files {
		err = os.Remove(config.Configuration.Paths.DatabaseDumps + "/" + file.Name())
		if err != nil {
			log.Errorf("error removing file %v", err)
		}
	}
}

func cleanWorkdir() {
	log.Debug("remove workdir")
	dir, err := os.Open(workdir)
	if err != nil {
		log.Errorf("error opening dir %v", err)
	}

	defer dir.Close()

	files, _ := dir.Readdir(0)
	for _, file := range files {
		os.Remove(config.Configuration.Paths.DatabaseDumps + "/" + file.Name())
		if err != nil {
			log.Errorf("Error removing dump %v", err)
		}
	}

}

func consolidateDatabaseDumps() {
	dir, err := os.Open(config.Configuration.Paths.DatabaseDumps)
	if err != nil {
		log.Errorf("error opening dir %v", err)
	}

	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		log.Errorf("error reading dir %v", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".sql" {
			err := os.Rename(config.Configuration.Paths.DatabaseDumps+"/"+file.Name(), dbdump+file.Name())
			if err != nil {
				log.Errorf("unable to move file: %v", err)
			}
		}
	}

}

func databaseDump(ctx context.Context, database string) {
	command := []string{"bash", "-c", "/usr/bin/mysqldump -u " + config.Configuration.Database.Credentials.Username + " --password=" + config.Configuration.Database.Credentials.Password + " " + database + " > /dump/" + database + ".sql"}
	log.Debugf("constructed docker exec command: %v", command)
	execConfig := types.ExecConfig{Tty: false, AttachStdout: true, AttachStderr: false, Cmd: command}
	respIdExecCreate, err := dc.ContainerExecCreate(ctx, "mysql", execConfig)
	if err != nil {
		log.Errorf("error creating db dump command: %v", err)
	}
	err = dc.ContainerExecStart(ctx, respIdExecCreate.ID, types.ExecStartCheck{})
	if err != nil {
		log.Errorf("error occured starting db dump: %v", err)
	}

	execStatus, err := dc.ContainerExecInspect(ctx, respIdExecCreate.ID)
	if err != nil {
		log.Errorf("error occured inspecting dump progress: %v", err)
	}
	for execStatus.Running {
		log.Info("waiting for db dump to finish...")
		time.Sleep(2 * time.Second)
	}
}

func compressDir(srcPath string, destFile string) error {
	cmd := exec.Command("tar", "-zcvf", destFile+".tar.gz", srcPath)
	err := cmd.Run()
	if err != nil {
		log.Errorf("unable to compress %v: %v", srcPath, err)
		return err
	}
	return nil
}

func main() {
	setup()
	defer cleanup()

	// Dump all databases
	for _, database := range config.Configuration.Database.Databases {
		log.Infof("dumping database %v", database)
		databaseDump(ctx, database) //TODO: Use goroutines to dump databases in parallel and make the backup more efficent
	}

	// Gather dumps in workdir
	consolidateDatabaseDumps()

	// Compress all data directories
	for _, path := range config.Configuration.Paths.FileDumps {
		log.Infof("compressing %v", path)
		compressDir(path, fsdump+filepath.Base(path)) //TODO: Use goroutines to compress filesystems in parallel and make the backup more efficent
	}

	// Compress full backup
	log.Info("compressing backup")

	compressDir(workdir, "/tmp/backup-"+timestamp)
	defer conn.Close()

	// Create new SFTP client
	sc, err := sftp.NewClient(conn)
	if err != nil {
		log.Errorf("unable to start SFTP subsystem: %v", err)
	}
	defer sc.Close()

	// Upload to SFTP site
	log.Info("uploading backup to store")
	uploadBackup(sc, "/tmp/backup-"+timestamp+".tar.gz", "backup-"+timestamp+".tar.gz")

	log.Info("done. exiting.")
}
