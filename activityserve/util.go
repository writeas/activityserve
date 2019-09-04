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

func formatJSON(theJSON []byte) string{
	dst := new(bytes.Buffer)
	json.Indent(dst, theJSON, "", "\t")
	return dst.String()
}
