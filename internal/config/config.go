package config

import (
	"fmt"
	"net"
	"os"

	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/spf13/viper"
)

var Configuration Config

type Config struct {
	Global       GlobalConfig
	Database     DatabaseConfig
	DataDirs     []string
	Paths        PathConfig
	DockerClient *client.Client
	SSHClient    *ssh.Client
	Sftp         SftpConfig
}

type GlobalConfig struct {
	Debug bool
}

type PathConfig struct {
	DatabaseDumps string
	FileDumps     []string
}

type DatabaseConfig struct {
	Credentials struct {
		Username string
		Password string
	}
	Databases []string
}

type SftpConfig struct {
	Url      string
	Port     int
	Username string
	Password string
}

func initViper() error {
	log.Debug("Reading config")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/jbackup")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("Config file not found: %v", err)
		} else {
			log.Fatalf(" Unknown error occured while reading config. error: %v", err)
		}
	}
	err := viper.Unmarshal(&Configuration)
	if err != nil {
		log.Fatalf("Error unmarshaling config: %v", err)
	}

	viper.WatchConfig()

	log.Infof("Using config file found at %v", viper.GetViper().ConfigFileUsed())

	return err
}

func initLogging() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	if Configuration.Global.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Enabled DEBUG logging level")
	}
}

func initDocker() *client.Client {
	log.Debugln("initializing Docker client")
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Panic(err)
	}
	return cli
}

func initSftp() {
	var err error
	fmt.Fprintf(os.Stdout, "Connecting to %v ...\n", Configuration.Sftp.Url)

	var auths []ssh.AuthMethod

	// Try to use $SSH_AUTH_SOCK which contains the path of the unix file socket that the sshd agent uses
	// for communication with other processes.
	if aconn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		auths = append(auths, ssh.PublicKeysCallback(agent.NewClient(aconn).Signers))
	}

	// Use password authentication if provided
	if Configuration.Sftp.Password != "" {
		auths = append(auths, ssh.Password(Configuration.Sftp.Password))
	}

	// Initialize client configuration
	config := ssh.ClientConfig{
		User: Configuration.Sftp.Username,
		Auth: auths,
		// Uncomment to ignore host key check
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		// HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	addr := fmt.Sprintf("%s:%d", Configuration.Sftp.Url, Configuration.Sftp.Port)
	fmt.Println(addr)

	// Connect to server
	Configuration.SSHClient, err = ssh.Dial("tcp", addr, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect to [%s]: %v\n", addr, err)
		os.Exit(1)
	}
}

func init() {

	// Build config
	err := initViper()
	if err != nil {
		log.Fatal("Unable to init config. Bye.")
	}

	// Configure logger
	initLogging()
	Configuration.DockerClient = initDocker()
	initSftp()
}
