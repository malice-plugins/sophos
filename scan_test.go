package main_test

import (
	"fmt"
	"strings"
	"testing"

	log "github.com/Sirupsen/logrus"
)

const resultString = `>>> Virus 'EICAR-AV-Test' found in file EICAR
`

const versionString = `SAVScan virus detection utility
Copyright (c) 1989-2017 Sophos Limited. All rights reserved.

System time 15:42:06, System date 16 June 2017

Product version           : 5.34.0
Engine version            : 3.68.0
Virus data version        : 5.40
User interface version    : 2.03.068
Platform                  : Linux/AMD64
Released                  : 30 May 2017
Total viruses (with IDEs) : 13427613
`

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

func parseSophosOutput(sophosout string) (string, error) {

	lines := strings.Split(sophosout, "\n")

	for _, line := range lines {
		if strings.Contains(line, ">>> Virus") && strings.Contains(line, "found in file") {
			parts := strings.Split(line, "'")
			fmt.Println(parts)
			return strings.TrimSpace(parts[1]), nil
		}
	}

	return "", nil
}

// TestParseResult tests the ParseFSecureOutput function.
func TestParseResult(t *testing.T) {

	results, err := parseSophosOutput(resultString)

	if err != nil {
		t.Log(err)
	}

	if true {
		t.Log("results: ", results)
	}

}

// TestParseVersion tests the GetFSecureVersion function.
func TestParseVersion(t *testing.T) {

	version, database := parseSophosVersion(versionString)

	if true {
		t.Log("version: ", version)
		t.Log("database: ", database)
	}

}
