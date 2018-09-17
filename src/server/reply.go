package server

import (
	"strconv"
	"strings"
	"fmt"
	"sync/atomic"
)

func AddReply(s *Server, c *Client, str string) {
	if c.PrepareClientToWrite() != C_OK {
		return
	}
	n, err := c.ReplyWriter.WriteString(str)
	if err != nil {
		return
	}
	atomic.AddInt64(&s.StatNetOutputBytes, int64(n))
}

func AddReplyStrObj(s *Server, c *Client, o *StrObject) {
	if !CheckOType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	if err == nil {
		AddReply(s, c, str)
	} else {
		return
	}
}

func AddReplyError(s *Server, c *Client, str string) {
	if len(str) != 0 || str[0] != '-' {
		AddReply(s, c, "-ERR ")
	}
	AddReply(s, c, str)
	AddReply(s, c, "\r\n")
}

func AddReplyErrorFormat(s *Server, c *Client, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	AddReplyError(s, c, str)
}

func AddReplyStatus(s *Server, c *Client, str string) {
	AddReply(s, c, "+")
	AddReply(s, c, str)
	AddReply(s, c, "\r\n")
}

func (s *Server) AddReplyStatusFormat(c *Client, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	AddReplyStatus(s, c, str)
}

//func (s *Server) AddReplyHelp(c *Client, help []string) {
//	cmd := c.Argv[0]
//	s.AddReplyStatusFormat(c, "%s <subcommand> arg arg ... arg. Subcommands are:", cmd)
//	for _, h := range help {
//		s.AddReplyStatus(c, h)
//	}
//}

func AddReplyIntWithPrifix(s *Server, c *Client, i int, prefix byte) {
	/* Things like $3\r\n or *2\r\n are emitted very often by the protocol
	so we have a few shared objects to use if the integer is small
	like it is most of the times. */
	if prefix == '*' && i >= 0 && i < SHARED_BULKHDR_LEN {
		AddReply(s, c, s.Shared.MultiBulkHDR[i])
	} else if prefix == '$' && i >= 0 && i < SHARED_BULKHDR_LEN {
		AddReply(s, c, s.Shared.BulkHDR[i])
	} else {
		str := strconv.Itoa(i)
		buf := Buffer{}
		buf.WriteByte(prefix)
		buf.WriteString(str)
		buf.WriteString("\r\n")
		AddReply(s, c, buf.String())
	}
}

func AddReplyInt(s *Server, c *Client, i int) {
	if i == 0 {
		AddReply(s, c, s.Shared.Zero)
	} else if i == 1 {
		AddReply(s, c, s.Shared.One)
	} else {
		AddReplyIntWithPrifix(s, c, i, ':')
	}
}

func AddReplyMultiBulkLen(s *Server, c *Client, length int) {
	AddReplyIntWithPrifix(s, c, length, '*')
}

/* Create the length prefix of a bulk reply, example: $2234 */
func AddReplyBulkLenOfStr(s *Server, c *Client, str string) {
	length := len(str)
	// fmt.Println(">>>>>>>>>>>>>>>>", length)
	AddReplyIntWithPrifix(s, c, length, '$')
}

func AddReplyBulkStrObj(s *Server, c *Client, o *StrObject) {
	if !CheckOType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	// fmt.Println(">>>>>>>>>>>>>>>>", str)
	if err == nil {
		AddReplyBulkLenOfStr(s, c, str)
	} else {
		return
	}
	AddReply(s, c, str)
	AddReply(s, c, s.Shared.Crlf)
}

func AddReplyBulkStr(s *Server, c *Client, str string) {
	if str == "" {
		AddReply(s, c, s.Shared.NullBulk)
	} else {
		AddReplyBulkLenOfStr(s, c, str)
		AddReply(s, c, str)
		AddReply(s, c, s.Shared.Crlf)
	}
}

func AddReplyBulkInt(s *Server, c *Client, i int) {
	str := strconv.Itoa(i)
	AddReplyBulkStr(s, c, str)
}

func AddReplySubcommandSyntaxError(s *Server, c *Client) {
	cmd := c.Argv[0]
	AddReplyErrorFormat(s, c, "Unknown subcommand or wrong number of arguments for '%s'. Try %s HELP.", cmd, strings.ToUpper(cmd))
}
