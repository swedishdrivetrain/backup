package sftp

import (
	"fmt"
	"net"
	"os"

	c "joomla-backup/internal/config"

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
	log.Tracef("sftp address: %v", addr)

	// Connect to server
	SSHClient, err := ssh.Dial("tcp", addr, &config)
	if err != nil {
		log.Errorf("failed to connect to [%s]: %v", addr, err)
		return nil, err
	}

	return SSHClient, nil
}
