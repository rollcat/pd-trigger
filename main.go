// https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgw-events-api-v2-overview
package main

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	pd "github.com/PagerDuty/go-pagerduty"
	"github.com/google/uuid"
	"github.com/timtadh/getopt"
	"gopkg.in/yaml.v3"
)

var debug = false

const (
	clientName = "pd-trigger"
	clientURL  = "https://github.com/rollcat/pd-trigger"
)

const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityError    = "error"
	SeverityInfo     = "info"
)

func helpUsage() {
	println(`Usage: pd-trigger [-h] [-k KEY] [-s SEVERITY] [other flags] <SUMMARY>`)
}

func helpFull() {
	helpUsage()
	print(`Flags:
    -h, --help      Print this help and exit
    --help-setup    Print help about setting up the config/integration
    --debug         Print verbose debug info useful for troubleshooting
    -c, --config    Specify alternative .yml configuration file path
                    (default: ~/.pd.yml, ~/.config/pagerduty.yml,
                     /etc/xdg/pagerduty.yml; per $XDG_CONFIG_*)
    -k, --key       Event deduplication key (default: random)
    -s, --severity  One of: c, w, e, i (critical, warning, error, info)
                    (default: info)
    -S, --source    Source (default: current hostname)
`)
}

func helpSetup() {
	print(`Setup:

1. Generate the auth token:
    - Open your Pagerduty dashboard
    - Integrations -> API Access Keys -> Create New API Key
    - Description: pd-trigger
    - Create Key

2. Generate the integration key:
    - Open your Pagerduty dashboard
    - Services -> Service Directory
    - Select a service, or create a new one
    - Integrations
    - "Events API V2" (create the integration if it does not exist)
    - Integration Key

3. Create ~/.config/pagerduty.yml (per XDG_CONFIG_HOME),
   or /etc/xdg/pagerduty.yml (per XDG_CONFIG_DIRS),
   with the following contents:

    authtoken: "<your auth token>"
    integrationkey: "<your integration key>"
`)
}

// check: https://github.com/PagerDuty/go-pagerduty/blob/v1.7.0/command/meta.go
type Config struct {
	AuthToken      string `yaml:"authtoken"`
	IntegrationKey string `yaml:"integrationkey"`
	// ignored; possibly for compatibility with go-pagerduty/command
	LogLevel string `yaml:"loglevel"`
}

func GetXdgConfigHome() string {
	d, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if ok {
		return d
	} else {
		return filepath.Join(os.Getenv("HOME"), ".config")
	}
}

func GetXdgConfigDirs() []string {
	ds, ok := os.LookupEnv("XDG_CONFIG_DIRS")
	if !ok {
		ds = "/etc/xdg"
	}
	return strings.Split(ds, ":")
}

func getConfig(fnames ...string) (config Config, err error) {
	for _, fname := range fnames {
		var f io.ReadCloser
		f, err = os.Open(fname)
		if err != nil {
			continue
		}
		defer f.Close()
		var data []byte
		data, err = io.ReadAll(f)
		if err != nil {
			continue
		}
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			continue
		}
		if debug {
			fmt.Printf("Using config file: %s\n", fname)
		}
		return
	}
	return
}

func main() {
	args, opts, err := getopt.GetOpt(
		os.Args[1:],
		"c:hk:s:S:",
		[]string{
			"config=",    // -c:
			"debug",      //
			"help",       // -h
			"help-setup", //
			"key=",       // -k:
			"severity=",  // -s:
			"source=",    // -S:
		},
	)
	if err != nil {
		println(err.Error())
		helpUsage()
	}

	hostname, _ := os.Hostname()
	var userName string
	if user, _ := user.Current(); user != nil {
		userName = user.Name
	}

	var summary string
	var source = hostname
	var severity = SeverityInfo
	var configFnames = []string{
		filepath.Join(os.Getenv("HOME"), ".pd.yml"),
		filepath.Join(GetXdgConfigHome(), "pagerduty.yml"),
	}
	for _, d := range GetXdgConfigDirs() {
		configFnames = append(configFnames, filepath.Join(d, "pagerduty.yml"))
	}
	var key string = uuid.New().String()

parseOpts:
	for _, opt := range opts {
		switch opt.Opt() {
		case "-c":
			fallthrough
		case "--config":
			configFnames = append([]string{opt.Arg()}, configFnames...)
		case "--debug":
			debug = true
		case "-h":
			fallthrough
		case "--help":
			helpFull()
			os.Exit(0)
		case "--help-setup":
			helpSetup()
			os.Exit(0)
		case "-k":
			fallthrough
		case "--key":
			key = opt.Arg()
		case "-s":
			fallthrough
		case "--severity":
			for _, s := range []string{
				SeverityCritical,
				SeverityWarning,
				SeverityError,
				SeverityInfo,
			} {
				if strings.HasPrefix(s, opt.Arg()) {
					severity = s
					continue parseOpts
				}
			}
			println(`Severity must be one of: critical, warning, error, info`)
			helpUsage()
			os.Exit(2)
		case "-S":
			fallthrough
		case "--source":
			source = opt.Arg()
		default:
			panic(opt.Opt())
		}
	}

	summary = strings.Join(args, " ")
	if summary == "" {
		helpUsage()
		os.Exit(2)
	}

	config, err := getConfig(configFnames...)
	if err != nil || config.AuthToken == "" || config.IntegrationKey == "" {
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
		}
		helpSetup()
		os.Exit(2)
	}

	client := pd.NewClient(config.AuthToken)
	if debug {
		client.SetDebugFlag(pd.DebugCaptureLastRequest | pd.DebugCaptureLastResponse)
	}
	ev := &pd.V2Event{
		RoutingKey: config.IntegrationKey,
		Action:     "trigger", // trigger, acknowledge, or resolve
		DedupKey:   key,
		Images:     []interface{}{},
		Links:      []interface{}{},
		Client:     clientName,
		ClientURL:  clientURL,
		Payload: &pd.V2Payload{
			Summary:   summary,
			Source:    source,
			Severity:  severity,
			Timestamp: time.Now().Format(time.RFC3339),
			Component: "",
			Group:     "",
			Class:     "",
			Details: map[string]string{
				"hostname": hostname,
				"username": userName,
			},
		},
	}
	evr, err := client.ManageEvent(ev)
	if debug {
		fmt.Printf("Response: %#v\n", evr)
	}
	if err != nil {
		// This is an incredibly unfortunate situation.
		fmt.Printf("Error:    %s\n", err.Error())
		fmt.Printf("          %#v\n", err)
		if rawr, ok := client.LastAPIResponse(); ok {
			fmt.Printf("RawResp:  %#v\n", rawr)
			bodyBytes, _ := io.ReadAll(rawr.Body)
			fmt.Printf("          %#v\n", string(bodyBytes))
		}
		println(`
This may indicate an error within this application, or (unlikely) an
issue with PagerDuty itself. Check <https://status.pagerduty.com> and
the issue tracker at <https://github.com/rollcat/pd-trigger/issues>.`)
		os.Exit(1)
	}
}
