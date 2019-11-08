package activityserve

import (
	"bufio"
	"io"
	"net/http"
	"os"

	// 	"net/url"
	"bytes"
	"encoding/json"

	// 	"time"
	// 	"fmt"
	"github.com/gologme/log"
	// 	"github.com/go-fed/httpsig"
)

func isSuccess(code int) bool {
	return code == http.StatusOK ||
		code == http.StatusCreated ||
		code == http.StatusAccepted
}

//PrettyPrint maps
func PrettyPrint(themap map[string]interface{}) {
	b, err := json.MarshalIndent(themap, "", "  ")
	if err != nil {
		log.Info("error:", err)
	}
	log.Print(string(b))
}

//PrettyPrintJSON does what it's name says
func PrettyPrintJSON(theJSON []byte) {
	dst := new(bytes.Buffer)
	json.Indent(dst, theJSON, "", "\t")
	log.Info(dst)
}

// FormatJSON formats json with tabs and
// returns the new string
func FormatJSON(theJSON []byte) string {
	dst := new(bytes.Buffer)
	json.Indent(dst, theJSON, "", "\t")
	return dst.String()
}

// FormatHeaders to string for printing
func FormatHeaders(header http.Header) string {
	buf := new(bytes.Buffer)
	header.Write(buf)
	return buf.String()
}

func context() [1]string {
	return [1]string{"https://www.w3.org/ns/activitystreams"}
}

// ReadLines reads specific lines from a file and returns them as
// an array of strings
func ReadLines(filename string, from, to int) (lines []string, err error) {
	lines = make([]string, 0, to-from)
	reader, err := os.Open(filename)
	if err != nil {
		log.Info("could not read file")
		log.Info(err)
		return
	}
	sc := bufio.NewScanner(reader)
	line := 0
	for sc.Scan() {
		line++
		if line >= from && line <= to {
			lines = append(lines, sc.Text())
		}
	}
	return lines, nil
}

func lineCounter(filename string) (int, error) {
	r, err := os.Open(filename)
	if err != nil {
		log.Info("could not read file")
		log.Info(err)
		return 0, nil
	}
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
