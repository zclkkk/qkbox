package provideripc

import (
	"io"

	"github.com/zclkkk/qkbox/internal/ipcframework"
)

func WriteFrame(w io.Writer, value interface{}) error {
	return ipcframework.WriteFrame(w, value)
}

func ReadFrame(r io.Reader, value interface{}) error {
	return ipcframework.ReadFrame(r, value)
}
