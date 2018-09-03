package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fatih/structs"
	"github.com/gorilla/mux"
	"github.com/malice-plugins/pkgs/database"
	"github.com/malice-plugins/pkgs/database/elasticsearch"
	"github.com/malice-plugins/pkgs/utils"
	"github.com/parnurzeal/gorequest"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	name     = "sophos"
	category = "av"
)

var (
	// Version stores the plugin's version
	Version string
	// BuildTime stores the plugin's build time
	BuildTime string

	path string

	// es is the elasticsearch database object
	es elasticsearch.Database
)

type pluginResults struct {
	ID   string      `json:"id" structs:"id,omitempty"`
	Data ResultsData `json:"sophos" structs:"sophos"`
}

// Sophos json object
type Sophos struct {
	Results ResultsData `json:"sophos"`
}

// ResultsData json object
type ResultsData struct {
	Infected bool   `json:"infected" structs:"infected"`
	Result   string `json:"result" structs:"result"`
	Engine   string `json:"engine" structs:"engine"`
	Database string `json:"database" structs:"database"`
	Updated  string `json:"updated" structs:"updated"`
	MarkDown string `json:"markdown,omitempty" structs:"markdown,omitempty"`
}

func assert(err error) {
	if err != nil {
		// Sophos exits with error status 3 if it finds a virus...why?!
		if err.Error() != "exit status 3" {
			log.WithFields(log.Fields{
				"plugin":   name,
				"category": category,
				"path":     path,
			}).Fatal(err)
		}
	}
}

// AvScan performs antivirus scan
func AvScan(timeout int) Sophos {

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var results ResultsData
	output, err := utils.RunCommand(ctx, "/opt/sophos/bin/savscan", "-f", "-ss", path)
	log.WithFields(log.Fields{
		"plugin":   name,
		"category": category,
		"path":     path,
	}).Debug("savscan Output: ", output)
	assert(err)

	if err != nil && err.Error() != "exit status 3" {
		// If fails try a second time
		output, err = utils.RunCommand(ctx, "/opt/sophos/bin/savscan", "-f", "-ss", path)
		log.WithFields(log.Fields{
			"plugin":   name,
			"category": category,
			"path":     path,
		}).Debug("savscan Output: ", output)
		assert(err)
	}

	results, err = ParseSophosOutput(output, err, path)
	assert(err)

	return Sophos{
		Results: results,
	}
}

// ParseSophosOutput convert sophos output into ResultsData struct
func ParseSophosOutput(sophosout string, err error, errpath string) (ResultsData, error) {

	// root@0e01fb905ffb:/malware# savscan -f EICAR
	// SAVScan virus detection utility
	// Version 5.21.0 [Linux/AMD64]
	// Virus data version 5.27, April 2016
	// Includes detection for 11283995 viruses, Trojans and worms
	// Copyright (c) 1989-2016 Sophos Limited. All rights reserved.
	//
	// System time 03:48:15, System date 22 August 2016
	// Command line qualifiers are: -f
	//
	// Full Scanning
	//
	// >>> Virus 'EICAR-AV-Test' found in file EICAR
	//
	// 1 file scanned in 4 seconds.
	// 1 virus was discovered.
	// 1 file out of 1 was infected.
	// If you need further advice regarding any detections please visit our
	// Threat Center at: http://www.sophos.com/en-us/threat-center.aspx
	// End of Scan.

	if err != nil && err.Error() != "exit status 3" {
		return ResultsData{}, err
	}

	version, database := getSophosVersion()

	sophos := ResultsData{
		Infected: false,
		Engine:   version,
		Database: database,
		Updated:  getUpdatedDate(),
	}

	lines := strings.Split(sophosout, "\n")

	for _, line := range lines {
		if strings.Contains(line, ">>> Virus") && strings.Contains(line, "found in file") {
			parts := strings.Split(line, "'")
			sophos.Result = strings.TrimSpace(parts[1])
			sophos.Infected = true
		}
	}

	return sophos, nil
}

// Get Anti-Virus scanner version
func getSophosVersion() (version string, database string) {
	// root@0e01fb905ffb:/malware# /opt/sophos/bin/savscan --version
	// SAVScan virus detection utility
	// Copyright (c) 1989-2016 Sophos Limited. All rights reserved.
	//
	// System time 03:41:05, System date 22 August 2016
	//
	// Product version           : 5.21.0
	// Engine version            : 3.64.0
	// Virus data version        : 5.27
	// User interface version    : 2.03.064
	// Platform                  : Linux/AMD64
	// Released                  : 26 April 2016
	// Total viruses (with IDEs) : 11283995
	versionOut, err := utils.RunCommand(nil, "/opt/sophos/bin/savscan", "--version")
	log.WithFields(log.Fields{
		"plugin":   name,
		"category": category,
		"path":     path,
	}).Debug("Sophos Version: ", versionOut)
	assert(err)

	return parseSophosVersion(versionOut)
}

func parseSophosVersion(versionOut string) (version string, database string) {

	lines := strings.Split(versionOut, "\n")

	for _, line := range lines {
		if strings.Contains(line, "Product version") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				version = strings.TrimSpace(parts[1])
			} else {
				log.Error("Umm... ", parts)
			}
		}
		if strings.Contains(line, "Virus data version") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				database = strings.TrimSpace(parts[1])
				break
			} else {
				log.Error("Umm... ", parts)
			}
		}
	}

	return
}

func parseUpdatedDate(date string) string {
	layout := "Mon, 02 Jan 2006 15:04:05 +0000"
	t, _ := time.Parse(layout, date)
	return fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())
}

func getUpdatedDate() string {
	if _, err := os.Stat("/opt/malice/UPDATED"); os.IsNotExist(err) {
		return BuildTime
	}
	updated, err := ioutil.ReadFile("/opt/malice/UPDATED")
	assert(err)
	return string(updated)
}

func updateAV(ctx context.Context) error {
	fmt.Println("Updating Sophos...")
	// root@0e01fb905ffb:/opt/sophos/update# ./savupdate.sh
	// Updating from versions - SAV: 9.12.1, Engine: 3.64.0, Data: 5.27
	// Updating Sophos Anti-Virus....
	// Updating Command-line programs
	// Updating SAVScan on-demand scanner
	// Updating sav-protect startup script
	// Updating sav-rms startup script
	// Updating Virus Engine and Data
	// Updating Sophos Anti-Virus Scanning Daemon
	// Updating Talpa Kernel Support
	// Updating Manifest
	// Selecting appropriate kernel support...
	// On-access scanning not available because of problems during kernel support compilation.
	// Update completed.
	// Updated to versions - SAV: 9.12.2, Engine: 3.65.2, Data: 5.30
	// Successfully updated Sophos Anti-Virus from sdds:SOPHOS
	output, err := utils.RunCommand(ctx, "/opt/sophos/update/savupdate.sh")
	log.WithFields(log.Fields{
		"plugin":   name,
		"category": category,
		"path":     path,
	}).Debug("Sophos update: ", output)
	assert(err)

	// Update UPDATED file
	t := time.Now().Format("20060102")
	err = ioutil.WriteFile("/opt/malice/UPDATED", []byte(t), 0644)
	return err
}

func generateMarkDownTable(s Sophos) string {
	var tplOut bytes.Buffer

	t := template.Must(template.New("sophos").Parse(tpl))

	err := t.Execute(&tplOut, s)
	if err != nil {
		log.Println("executing template:", err)
	}

	return tplOut.String()
}

func printStatus(resp gorequest.Response, body string, errs []error) {
	fmt.Println(body)
}

func webService() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/scan", webAvScan).Methods("POST")
	log.Info("web service listening on port :3993")
	log.Fatal(http.ListenAndServe(":3993", router))
}

func webAvScan(w http.ResponseWriter, r *http.Request) {

	r.ParseMultipartForm(32 << 20)
	file, header, err := r.FormFile("malware")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Please supply a valid file to scan.")
		log.Error(err)
	}
	defer file.Close()

	log.Debug("Uploaded fileName: ", header.Filename)

	tmpfile, err := ioutil.TempFile("/malware", "web_")
	if err != nil {
		assert(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	data, err := ioutil.ReadAll(file)
	assert(err)

	if _, err = tmpfile.Write(data); err != nil {
		assert(err)
	}
	if err = tmpfile.Close(); err != nil {
		assert(err)
	}

	// Do AV scan
	path = tmpfile.Name()
	sophos := AvScan(60)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(sophos); err != nil {
		assert(err)
	}
}

func main() {

	cli.AppHelpTemplate = utils.AppHelpTemplate
	app := cli.NewApp()

	app.Name = name
	app.Author = "blacktop"
	app.Email = "https://github.com/blacktop"
	app.Version = Version + ", BuildTime: " + BuildTime
	app.Compiled, _ = time.Parse("20060102", BuildTime)
	app.Usage = "Malice Sophos AntiVirus Plugin"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, V",
			Usage: "verbose output",
		},
		cli.StringFlag{
			Name:        "elasticsearch",
			Value:       "",
			Usage:       "elasticsearch url for Malice to store results",
			EnvVar:      "MALICE_ELASTICSEARCH_URL",
			Destination: &es.URL,
		},
		cli.BoolFlag{
			Name:  "table, t",
			Usage: "output as Markdown table",
		},
		cli.BoolFlag{
			Name:   "callback, c",
			Usage:  "POST results to Malice webhook",
			EnvVar: "MALICE_ENDPOINT",
		},
		cli.BoolFlag{
			Name:   "proxy, x",
			Usage:  "proxy settings for Malice webhook endpoint",
			EnvVar: "MALICE_PROXY",
		},
		cli.IntFlag{
			Name:   "timeout",
			Value:  60,
			Usage:  "malice plugin timeout (in seconds)",
			EnvVar: "MALICE_TIMEOUT",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "update",
			Aliases: []string{"u"},
			Usage:   "Update virus definitions",
			Action: func(c *cli.Context) error {
				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}
				return updateAV(nil)
			},
		},
		{
			Name:  "web",
			Usage: "Create a Sophos scan web service",
			Action: func(c *cli.Context) error {
				if c.GlobalBool("verbose") {
					log.SetLevel(log.DebugLevel)
				}
				webService()
				return nil
			},
		},
	}
	app.Action = func(c *cli.Context) error {

		var err error

		if c.Bool("verbose") {
			log.SetLevel(log.DebugLevel)
		}

		if c.Args().Present() {
			path, err = filepath.Abs(c.Args().First())
			assert(err)

			if _, err = os.Stat(path); os.IsNotExist(err) {
				assert(err)
			}

			sophos := AvScan(c.Int("timeout"))
			sophos.Results.MarkDown = generateMarkDownTable(sophos)

			// upsert into Database
			if len(c.String("elasticsearch")) > 0 {
				err := es.Init()
				if err != nil {
					return errors.Wrap(err, "failed to initalize elasticsearch")
				}
				err = es.StorePluginResults(database.PluginResults{
					ID:       utils.Getopt("MALICE_SCANID", utils.GetSHA256(path)),
					Name:     name,
					Category: category,
					Data:     structs.Map(sophos.Results),
				})
				if err != nil {
					return errors.Wrapf(err, "failed to index malice/%s results", name)
				}
			}

			if c.Bool("table") {
				fmt.Println(sophos.Results.MarkDown)
			} else {
				sophos.Results.MarkDown = ""
				sophosJSON, err := json.Marshal(sophos)
				assert(err)
				if c.Bool("post") {
					request := gorequest.New()
					if c.Bool("proxy") {
						request = gorequest.New().Proxy(os.Getenv("MALICE_PROXY"))
					}
					request.Post(os.Getenv("MALICE_ENDPOINT")).
						Set("X-Malice-ID", utils.Getopt("MALICE_SCANID", utils.GetSHA256(path))).
						Send(string(sophosJSON)).
						End(printStatus)

					return nil
				}
				fmt.Println(string(sophosJSON))
			}
		} else {
			log.Fatal(fmt.Errorf("please supply a file to scan with malice/sophos"))
		}
		return nil
	}

	err := app.Run(os.Args)
	assert(err)
}
