package main

//TODO: Complete log statements
import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"joomla-backup/internal/config"
	"joomla-backup/internal/sftp"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
)

var (
	workdir   = "/tmp/backup/"
	layoutISO = "2006-01-02T15:04:05"
	dbdump    = workdir + "db/"
	fsdump    = workdir + "fs/"
	dc        = config.Configuration.DockerClient
	timestamp string
	timezone  *time.Location
	ctx       context.Context
)

func setup() {
	var err error
	// set timestamp of backup
	timezone, err = time.LoadLocation(config.Configuration.Global.Timezone)
	if err != nil {
		log.Fatalf("Error loading timezone. Is the format correct?")
	}

	time := time.Now().In(timezone)
	timestamp = time.Format(layoutISO)
	ctx = context.Background()

	log.Debugf("creating workdir %v", workdir)
	os.Mkdir(workdir, 0755)
	log.Debugf("creating database dump folder %v", dbdump)
	os.Mkdir(dbdump, 0755)
	log.Debugf("creating filesystem dump folder %v", fsdump)
	os.Mkdir(fsdump, 0755)
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

func cleanupOldBackups() error {

	files, err := sftp.ListBackups()
	if err != nil {
		log.Fatalf("error in listing backups: %v", err)
	}

	for i := range files {
		if files[i].ModTime().Before(time.Now().In(timezone).AddDate(0, -config.Configuration.Global.MaxAge, 0)) {

			switch config.Configuration.Global.Dryrun {
			case false:
				sftp.DeleteBackup(files[i].Name())
			default:
				log.Infof("simulating delete of file %s", files[i].Name())
			}

		}
	}

	return nil
}

func main() {
	setup()
	defer cleanupOldBackups()
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

	// Upload to SFTP site
	log.Info("uploading backup to store")
	sftp.UploadBackup("/tmp/backup-"+timestamp+".tar.gz", "backup-"+timestamp+".tar.gz")

	log.Info("done. exiting.")
}
