package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// Format represents the output format.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatYAML  Format = "yaml"
)

// Error represents a structured error response.
type Error struct {
	Code    string `json:"code" yaml:"code"`
	Message string `json:"message" yaml:"message"`
}

// ErrorResponse wraps an error for output.
type ErrorResponse struct {
	Error Error `json:"error" yaml:"error"`
}

// Print outputs data in the specified format to stdout.
func Print(format Format, data any) error {
	return Fprint(os.Stdout, format, data)
}

// Fprint outputs data in the specified format to the given writer.
func Fprint(w io.Writer, format Format, data any) error {
	switch format {
	case FormatTable:
		return printTable(w, data)
	case FormatYAML:
		return printYAML(w, data)
	default:
		return printJSON(w, data)
	}
}

// PrintError outputs a structured error.
func PrintError(format Format, code, message string) {
	resp := ErrorResponse{
		Error: Error{Code: code, Message: message},
	}
	Print(format, resp)
}

func printJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func printYAML(w io.Writer, data any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	return enc.Encode(data)
}

func printTable(w io.Writer, data any) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	v := reflect.ValueOf(data)

	// If it's a slice, print each element as a row
	if v.Kind() == reflect.Slice {
		if v.Len() == 0 {
			fmt.Fprintln(tw, "No results")
			return nil
		}
		// Print headers from first element
		first := indirect(v.Index(0))
		if first.Kind() == reflect.Map {
			return printMapSliceTable(tw, v)
		}
		if first.Kind() == reflect.Struct {
			return printStructSliceTable(tw, v)
		}
	}

	// Single map
	if v.Kind() == reflect.Map {
		return printMapTable(tw, v)
	}

	// Single struct
	v = indirect(v)
	if v.Kind() == reflect.Struct {
		return printStructTable(tw, v)
	}

	// Fallback to JSON
	return printJSON(w, data)
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	return v
}

func printMapTable(tw *tabwriter.Writer, v reflect.Value) error {
	for _, key := range v.MapKeys() {
		fmt.Fprintf(tw, "%v\t%v\n", key, v.MapIndex(key))
	}
	return nil
}

func printMapSliceTable(tw *tabwriter.Writer, v reflect.Value) error {
	first := indirect(v.Index(0))
	keys := first.MapKeys()
	headers := make([]string, len(keys))
	for i, k := range keys {
		headers[i] = strings.ToUpper(fmt.Sprintf("%v", k))
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	for i := 0; i < v.Len(); i++ {
		elem := indirect(v.Index(i))
		vals := make([]string, len(keys))
		for j, k := range keys {
			vals[j] = fmt.Sprintf("%v", elem.MapIndex(k))
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	return nil
}

func printStructSliceTable(tw *tabwriter.Writer, v reflect.Value) error {
	first := indirect(v.Index(0))
	t := first.Type()

	headers := make([]string, 0, t.NumField())
	indices := make([]int, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Tag.Get("json")
		if name == "" || name == "-" {
			name = f.Name
		}
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}
		headers = append(headers, strings.ToUpper(name))
		indices = append(indices, i)
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	for i := 0; i < v.Len(); i++ {
		elem := indirect(v.Index(i))
		vals := make([]string, len(indices))
		for j, idx := range indices {
			vals[j] = formatValue(elem.Field(idx))
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	return nil
}

func printStructTable(tw *tabwriter.Writer, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Tag.Get("json")
		if name == "" || name == "-" {
			name = f.Name
		}
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}
		fmt.Fprintf(tw, "%s\t%s\n", strings.ToUpper(name), formatValue(v.Field(i)))
	}
	return nil
}

// formatValue dereferences pointers and formats nil as "-".
func formatValue(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "-"
		}
		return fmt.Sprintf("%v", v.Elem())
	}
	return fmt.Sprintf("%v", v)
}
