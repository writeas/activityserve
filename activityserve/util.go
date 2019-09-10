package activityserve

import (
	"net/http"
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
