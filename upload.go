package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"

	c "github.com/AnimeNL/joomla-backup/internal/config"
	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func initSftp() (*ssh.Client, error) {
	var err error
	log.Infof("connecting to %v ...", c.Configuration.Sftp.Url)

	var auths []ssh.AuthMethod

	// Try to use $SSH_AUTH_SOCK which contains the path of the unix file socket that the sshd agent uses
	// for communication with other processes.
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
	}

	// Use password authentication if provided
	if c.Configuration.Sftp.Password != "" {
		auths = append(auths, ssh.Password(c.Configuration.Sftp.Password))
	}

	// Initialize client configuration
	config := ssh.ClientConfig{
		User: c.Configuration.Sftp.Username,
		Auth: auths,
		// Uncomment to ignore host key check
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	addr := fmt.Sprintf("%s:%d", c.Configuration.Sftp.Url, c.Configuration.Sftp.Port)
	log.Debugf("sftp address: %v", addr)

	// Connect to server
	SSHClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		log.Errorf("failed to connect to [%s]: %v", addr, err)
		return nil, err
	}

	return SSHClient, nil
}

// Upload file to sftp server
func uploadBackup(localFile, remoteFile string) (err error) {
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
