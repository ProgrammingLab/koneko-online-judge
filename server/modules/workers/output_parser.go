package workers

import (
	"io"

	"github.com/gedorinku/koneko-online-judge/server/logger"
)

type OutputParser struct {
	output    io.Reader
	separator string
	next      string
	cur       int
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
	remain := 0 < len(p.next)

	for {
		buf := make([]byte, bufLen)
		n, err := p.output.Read(buf)

		if err != nil && err != io.EOF {
			logger.AppLog.Error(err)
			return false, "", err
		}
		p.eof = err == io.EOF

		buf = buf[:n]
		if remain {
			buf = append([]byte(p.next), buf...)
			p.next = ""
			remain = false
		}
		bufStr := string(buf)

		for i, b := range buf {
			if p.separator[p.cur] != b {
				p.cur = 0
				if p.separator[0] == b {
					p.cur = 1
				}
				continue
			}

			if p.cur < spLen-1 {
				p.cur++
				continue
			}

			res := p.next + bufStr[:i+1]
			p.next = bufStr[i+1:]
			k := len(res) - spLen
			p.cur = 0
			return !p.eof, res[:k], nil
		}
		if p.eof {
			return false, p.next + bufStr, nil
		}
		p.next += bufStr
	}
}
