package workers

import (
	"io"
	"strings"

	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type OutputParser struct {
	output    io.Reader
	separator string
	next      string
	eof       bool
}

func newReaderParser(reader io.Reader, separator string) OutputParser {
	return OutputParser{
		output:    reader,
		separator: separator,
		next:      "",
	}
}

func (p *OutputParser) Next() (bool, string, error) {
	spLen := len(p.separator)
	bufLen := 2 * len(p.separator)
	step := spLen
	cur := 0

	for {
		buf := make([]byte, bufLen)
		n, err := p.output.Read(buf)

		if err != nil && err != io.EOF {
			logger.AppLog.Error(err)
			return false, "", err
		}
		p.eof = err == io.EOF || n != bufLen

		buf = buf[:n]
		p.next += string(buf)

		i := strings.Index(p.next[cur:], p.separator)
		if i == -1 {
			if p.eof {
				res := p.next
				p.next = ""
				return false, res, nil
			}
			cur += step
			if step == spLen {
				step += spLen
			}
			continue
		}
		i += cur
		res := p.next[:i]
		p.next = p.next[i+spLen:]
		return true, res, nil
	}
}
