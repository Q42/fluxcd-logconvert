package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/mickep76/mapslice-json"
	"github.com/msoap/byline"
)

var outputToStderr bool

func main() {
	flag.BoolVar(&outputToStderr, "stderr", false, "if set, filtered output is send to stderr instad of stdout")
	flag.Parse()

	var dest io.Writer
	if outputToStderr {
		dest = os.Stderr
	} else {
		dest = os.Stdout
	}
	_, err := io.Copy(dest, convert(os.Stdin))
	if err != nil {
		os.Exit(1)
	}
}

func convert(reader io.Reader) io.Reader {
	// Create new line-by-line Reader from io.Reader &
	// add to the Reader stack of a filter functions
	return byline.NewReader(reader).MapErr(convertFluxLogLine)
}

func convertFluxLogLine(data []byte) (result []byte, err error) {
	// Output non-JSON lines as-is
	if nospace := bytes.TrimSpace(data); len(nospace) > 0 && nospace[0] != '{' {
		return data, nil
	}

	// Read JSON logline
	dst := map[string]interface{}{}
	if err := json.Unmarshal(data, &dst); err != nil {
		return nil, err
	}

	// Defaults
	var severity = "DEBUG"
	message, _ := dst["msg"].(string)

	// Parse different types of Flux json logs:
	// { "err": "some error" }
	// { "err": null, "output": "some info" }
	// { "msg": "some info" }
	// { "info": "some info" }
	var hasField bool
	var m interface{}
	if m, hasField = dst["err"]; hasField {
		severity = "ERROR"
		message, _ = m.(string)
	} else if m, hasField = dst["msg"]; hasField {
		severity = "INFO"
		message, _ = m.(string)
	} else if m, hasField = dst["info"]; hasField {
		severity = "INFO"
		message, _ = m.(string)
	} else if m, hasField = dst["output"]; hasField {
		severity = "INFO"
		message, _ = m.(string)
	}

	var dynamicFields = mapslice.MapSlice{}
	// Stable sorted keys in JSON output (for snapshot testability)
	var ks = keys(dst)
	sort.Strings(ks)
	for _, k := range ks {
		if k != "ts" {
			dynamicFields = append(dynamicFields, mapslice.MapItem{Key: k, Value: dst[k]})
		}
	}

	// Output log according to Google LogEntry/Stackdriver specs
	fields := append(mapslice.MapSlice{
		mapslice.MapItem{Key: "severity", Value: severity},
		mapslice.MapItem{Key: "timestamp", Value: dst["ts"]},
		mapslice.MapItem{Key: "message", Value: message},
		mapslice.MapItem{Key: "serviceContext", Value: serviceContext{"fluxcd"}},
	}, dynamicFields...)

	if sourceLoc := parseCaller(dst["caller"]); sourceLoc.Key != nil {
		fields = append(fields, sourceLoc)
	}

	result, err = json.Marshal(fields)
	return append(result, '\r', '\n'), err
}

func parseCaller(c interface{}) mapslice.MapItem {
	caller, isString := c.(string)
	if !isString {
		return mapslice.MapItem{Key: nil, Value: nil}
	}
	parts := strings.SplitN(caller, ":", 2)
	if len(parts) < 2 {
		return mapslice.MapItem{Key: nil, Value: nil}
	}
	return mapslice.MapItem{Key: "logging.googleapis.com/sourceLocation", Value: sourceLocation{parts[0], parts[1], "unknown"}}
}

func keys(m map[string]interface{}) (result []string) {
	for k := range m {
		result = append(result, k)
	}
	return
}

type sourceLocation struct {
	File     string `json:"file,omitempty"`
	Line     string `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

type serviceContext struct {
	Service string `json:"service"`
}
