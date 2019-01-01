package server

import (
	"fmt"
)

func (s *Server) ServerLogDebugF(format string, a ...interface{}) {
	if s.LogLevel >= LL_DEBUG {
		fmt.Printf(format, a...)
	}
}

func (s *Server) ServerLogInfoF(format string, a ...interface{}) {
	if s.LogLevel >= LL_INFO {
		fmt.Printf(format, a...)
	}
}

func (s *Server) ServerLogNoticeF(format string, a ...interface{}) {
	if s.LogLevel >= LL_NOTICE {
		fmt.Printf(format, a...)
	}
}

func (s *Server) ServerLogWarnF(format string, a ...interface{}) {
	if s.LogLevel >= LL_WARNING {
		fmt.Printf(format, a...)
	}
}

func (s *Server) ServerLogErrorF(format string, a ...interface{}) {
	fmt.Errorf(format, a...)
}





