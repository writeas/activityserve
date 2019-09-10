package activityserve

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gologme/log"
	"gopkg.in/ini.v1"
)

var slash = string(os.PathSeparator)
var baseURL = "http://example.com/"
var storage = "storage"
var userAgent = "activityserve"
var printer *log.Logger

const libName = "activityserve"
const version = "0.99"

var client = http.Client{}

// Setup sets our environment up
func Setup(configurationFile string, debug bool) {
	// read configuration file (config.ini)

	if configurationFile == "" {
		configurationFile = "config.ini"
	}

	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	// Load base url from configuration file
	baseURL = cfg.Section("general").Key("baseURL").String()
	// check if it ends with a / and append one if not
	if baseURL[len(baseURL)-1:] != "/" {
		baseURL += "/"
	}
	// print it for our users
	fmt.Println()
	fmt.Println("Domain Name:", baseURL)

	// Load storage location (only local filesystem supported for now) from config
	storage = cfg.Section("general").Key("storage").String()
	cwd, err := os.Getwd()
	fmt.Println("Storage Location:", cwd+slash+storage)
	fmt.Println()

	SetupStorage(storage)

	// Load user agent
	userAgent = cfg.Section("general").Key("userAgent").String()

	// I prefer long file so that I can click it in the terminal and open it
	// in the editor above
	log.SetFlags(log.Llongfile)
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.EnableLevel("warn")
	// create a logger with levels but without prefixes for easier to read
	// debug output
	printer = log.New(os.Stdout, " ", 0)

	if debug == true {
		fmt.Println()
		fmt.Println("debug mode on")
		log.EnableLevel("info")
		printer.EnableLevel("info")
	}
}

// SetupStorage creates storage
func SetupStorage(storage string) {
	// prepare storage for foreign activities (activities we store that don't
	// belong to us)
	foreignDir := storage + slash + "foreign"
	if _, err := os.Stat(foreignDir); os.IsNotExist(err) {
		os.MkdirAll(foreignDir, 0755)
	}
}
