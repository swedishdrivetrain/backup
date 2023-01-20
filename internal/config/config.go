package config

import (
	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

var Configuration Config

type Config struct {
	Global       GlobalConfig
	Database     DatabaseConfig
	DataDirs     []string
	Paths        PathConfig
	DockerClient *client.Client
	Sftp         SftpConfig
}

type GlobalConfig struct {
	Debug  bool
	Dryrun bool
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
			log.Fatalf("config file not found: %v", err)
		} else {
			log.Fatalf("unknown error occured while reading config. error: %v", err)
		}
	}
	err := viper.Unmarshal(&Configuration)
	if err != nil {
		log.Fatalf("error unmarshaling config: %v", err)
	}

	viper.WatchConfig()

	log.Infof("using config file found at %v", viper.GetViper().ConfigFileUsed())

	return err
}

func initLogging() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	if Configuration.Global.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("enabled DEBUG logging level")
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

func init() {

	// Build config
	err := initViper()
	if err != nil {
		log.Fatal("unable to init config. Bye.")
	}

	// Check if an ssh password is provided for sftp
	if Configuration.Sftp.Password == "" {
		log.Fatal("No password provided for SFTP. Cannot run.")
	}

	// Configure logger
	initLogging()
	Configuration.DockerClient = initDocker()
}
