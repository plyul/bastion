package log

import (
	"bytes"
	"go.uber.org/zap"
	"regexp"
)

const bufferSize = 1024

type BufferedLogger struct {
	baseLogger *zap.Logger
	streamID   string
	buffer     *bytes.Buffer
}

func NewBufferedLogger(baseLogger *zap.Logger, streamID string) BufferedLogger {
	bl := BufferedLogger{
		baseLogger: baseLogger,
		streamID:   streamID,
		buffer:     bytes.NewBuffer(make([]byte, bufferSize)),
	}
	bl.buffer.Reset()
	return bl
}

func (bl *BufferedLogger) Write(p []byte) (n int, err error) {
	controlSequences := regexp.MustCompile(`\x1b[ #%()*+\-.\/].|(?:\x1b\[|\x9b)[ -?]*[@-~]|(?:\x1b\]|\x9d).*?(?:\x1b\\|[\a\x9c])|(?:\x1b[P^_]|[\x90\x9e\x9f]).*?(?:\x1b\\|\x9c)|\x1b.|[\x80-\x9f]`)
	filteredData := controlSequences.ReplaceAll(p, []byte(""))
	newLine := regexp.MustCompile("(\r\n|\r|\n)")
	lines := newLine.Split(string(filteredData), -1)
	for _, line := range lines {
		n, err := bl.buffer.Write([]byte(line))
		if err != nil {
			return n, err
		}
		l := bl.buffer.Len()
		if l > 0 {
			lb := bl.buffer.Bytes()[l-1]
			if lb == '\r' {
				bl.buffer.Truncate(l - 1)
			}
		}
		if len(lines) == 1 { // переданный массив не содержит \n, просто добавляем его к буферу
			return n, nil
		}
		if bl.buffer.Len() > 0 {
			bl.baseLogger.Info(bl.buffer.String(), zap.String("stream_id", bl.streamID))
		}
		bl.buffer.Reset()
	}
	return 0, nil
}

func (bl *BufferedLogger) Close() error {
	bl.baseLogger.Info(bl.buffer.String(), zap.String("stream_id", bl.streamID))
	return bl.baseLogger.Sync()
}
