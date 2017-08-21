package template

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/oklog/ulid"
	"github.com/wallix/awless/template/internal/ast"
)

type renderFunc func(...interface{}) string

func renderNoop(s ...interface{}) string { return fmt.Sprint(s) }

type Printer interface {
	Print(*TemplateExecution) error
}

func NewLogPrinter(w io.Writer) *logPrinter {
	return &logPrinter{
		w:        w,
		RenderOK: renderNoop,
		RenderKO: renderNoop,
	}
}

func NewDefaultPrinter(w io.Writer) *defaultPrinter {
	return &defaultPrinter{
		w:        w,
		RenderOK: renderNoop,
		RenderKO: renderNoop,
	}
}

func NewJSONPrinter(w io.Writer) *jsonPrinter {
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return &jsonPrinter{
		enc: enc,
	}
}

type defaultPrinter struct {
	w io.Writer

	RenderOK renderFunc
	RenderKO renderFunc
}

func (p *defaultPrinter) Print(t *TemplateExecution) error {
	buff := bufio.NewWriter(p.w)

	tabw := tabwriter.NewWriter(buff, 0, 8, 0, '\t', 0)
	for _, expr := range t.expressionNodesIterator() {
		var action, entity string
		switch e := expr.(type) {
		case *ast.CommandNode:
			action = e.Action
			entity = e.Entity
		case *ast.ActionNode:
			action = e.Action
			entity = e.Entity
		}

		var status string

		if expr.Err() != nil {
			status = p.RenderKO("KO")
		} else {
			status = p.RenderOK("OK")
		}

		var line string
		if v, ok := expr.Result().(string); ok && v != "" {
			line = fmt.Sprintf("    %s\t%s = %s\t", status, entity, v)
		} else {
			line = fmt.Sprintf("    %s\t%s %s\t", status, action, entity)
		}

		fmt.Fprintln(tabw, line)
		if expr.Err() != nil {
			for _, err := range formatMultiLineErrMsg(expr.Err().Error()) {
				fmt.Fprintf(tabw, "%s\t%s\n", "", err)
			}
		}
	}

	tabw.Flush()
	buff.Flush()

	return nil
}

type logPrinter struct {
	w io.Writer

	RenderOK renderFunc
	RenderKO renderFunc
}

func (p *logPrinter) Print(t *TemplateExecution) error {
	buff := bufio.NewWriter(p.w)

	buff.WriteString(fmt.Sprintf("ID: %s\tDate: %s", t.ID, parseULIDDate(t.ID)))
	if t.Author != "" {
		buff.WriteString(fmt.Sprintf("\tAuthor: %s", t.Author))
	}
	if t.Locale != "" {
		buff.WriteString(fmt.Sprintf("\tRegion: %s", t.Locale))
	}
	if t.Profile != "" {
		buff.WriteString(fmt.Sprintf("\tProfile: %s", t.Profile))
	}
	if !IsRevertible(t.Template) {
		buff.WriteString(" (not revertible)")
	}
	buff.WriteString("\n")

	tabw := tabwriter.NewWriter(buff, 0, 8, 0, '\t', 0)
	for _, cmd := range t.CommandNodesIterator() {
		var result, status string

		exec := fmt.Sprintf("%s", cmd.String())

		if cmd.CmdErr != nil {
			status = p.RenderKO("KO")
		} else {
			status = p.RenderOK("OK")
		}

		if v, ok := cmd.CmdResult.(string); ok && v != "" {
			result = fmt.Sprintf("[%s]", v)
		}

		line := fmt.Sprintf("    %s\t%s\t%s\t", status, exec, result)

		fmt.Fprintln(tabw, line)
		if cmd.CmdErr != nil {
			for _, err := range formatMultiLineErrMsg(cmd.CmdErr.Error()) {
				fmt.Fprintf(tabw, "%s\t%s\n", "", err)
			}
		}

	}

	tabw.Flush()
	buff.Flush()

	return nil
}

type jsonPrinter struct {
	enc *json.Encoder
}

func (p *jsonPrinter) Print(t *TemplateExecution) error {
	if err := p.enc.Encode(t); err != nil {
		return fmt.Errorf("json printer: %s", err)
	}
	return nil
}

func formatMultiLineErrMsg(msg string) []string {
	notabs := strings.Replace(msg, "\t", "", -1)
	var indented []string
	for _, line := range strings.Split(notabs, "\n") {
		indented = append(indented, fmt.Sprintf("    %s", line))
	}
	return indented
}

func parseULIDDate(uid string) string {
	parsed, err := ulid.Parse(uid)
	if err != nil {
		panic(err)
	}

	date := time.Unix(int64(parsed.Time())/int64(1000), time.Nanosecond.Nanoseconds())

	return date.Format(time.Stamp)
}
