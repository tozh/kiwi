package networking

import (
	"fmt"
	"syscall"
)

func AnetSetErrorFormat(err *string, format string, a ...interface{}) {
	if err == nil  {
		return
	}
	*err = fmt.Sprintf(format, a)
}

func AnetSetBlock(err *string, fd int64, nonBlock bool) {

}