package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"sort"
	"strconv"
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
	if message, hasField = dst["warn"].(string); hasField {
		severity = "WARNING"
	} else if message, hasField = dst["err"].(string); hasField {
		severity = "ERROR"
	} else if message, hasField = dst["msg"].(string); hasField {
		severity = "INFO"
	} else if message, hasField = dst["info"].(string); hasField {
		severity = "INFO"
	} else if message, hasField = dst["output"].(string); hasField {
		severity = "INFO"
	} else {
		message = queryFormat(dst)
	}

	var dynamicFields = mapslice.MapSlice{}
	mapSortedForEach(dst, func(k string, value interface{}) {
		if k != "ts" && k != "caller" {
			dynamicFields = append(dynamicFields, mapslice.MapItem{Key: k, Value: value})
		}
	})

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

// Stable sorting (for snapshot testability)
func mapSortedForEach(m map[string]interface{}, fn func(key string, value interface{})) {
	var ks = keys(m)
	sort.Strings(ks)
	for _, k := range ks {
		fn(k, m[k])
	}
}

func queryFormat(m map[string]interface{}) string {
	b := strings.Builder{}
	mapSortedForEach(m, func(k string, v interface{}) {
		if strValue, isString := v.(string); k != "ts" && k != "caller" && isString {
			if b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString(k)
			b.WriteString(`=`)
			b.WriteString(strconv.Quote(strValue))
		}
	})
	return b.String()
}

type sourceLocation struct {
	File     string `json:"file,omitempty"`
	Line     string `json:"line,omitempty"`
	Function string `json:"function,omitempty"`
}

type serviceContext struct {
	Service string `json:"service"`
}
