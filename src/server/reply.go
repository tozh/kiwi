package server

import (
	"strconv"
	"strings"
	"fmt"
	"bytes"
)

func AddReplyToBuffer(s *Server, c *Client, str string) int64 {
	if c.ReplyList.ListLength() > 0 {
		return C_ERR
	}
	available := len(c.ReplyBuf) - int(c.ReplyBufSize)
	if len(str) > available {
		return C_ERR
	}
	copy(c.ReplyBuf[c.ReplyBufSize:], str)
	c.ReplyBufSize += int64(len(str))
	return C_OK
}

func AddReplyToList(s *Server, c *Client, str string) {
	if c.ReplyList.ListLength() == 0 {
		c.ReplyList.ListAddNodeTail(&str)
		c.ReplyListSize += int64(len(str))
	} else {
		node := c.ReplyList.ListTail()
		tail := *node.Value.(*string)
		if tail != "" && len(tail)+len(str) < int(s.ClientMaxReplyBufLen) {
			tail = CatString(tail, str)
			node.Value = &tail
			c.ReplyListSize += int64(len(str))
		} else {
			c.ReplyList.ListAddNodeTail(&str)
			c.ReplyListSize += int64(len(str))
		}
	}
}

func AddReply(s *Server, c *Client, str string) {
	if c.PrepareClientToWrite() != C_OK {
		return
	}
	if AddReplyToBuffer(s, c, str) == C_ERR {
		AddReplyToList(s, c, str)
	}
}

func AddReplyStrObj(s *Server, c *Client, o *StrObject) {
	if !CheckRType(o, OBJ_RTYPE_STR) {
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

func AddReplyIntWithPrifix(s *Server, c *Client, i int64, prefix byte) {
	/* Things like $3\r\n or *2\r\n are emitted very often by the protocol
	so we have a few shared objects to use if the integer is small
	like it is most of the times. */
	if prefix == '*' && i >= 0 && i < SHARED_BULKHDR_LEN {
		AddReply(s, c, s.Shared.MultiBulkHDR[i])
	} else if prefix == '$' && i >= 0 && i < SHARED_BULKHDR_LEN {
		AddReply(s, c, s.Shared.MultiBulkHDR[i])
	} else {
		str := strconv.FormatInt(i, 10)
		buf := bytes.Buffer{}
		buf.WriteByte(prefix)
		buf.WriteString(str)
		buf.WriteByte('\r')
		buf.WriteByte('\n')
		AddReply(s, c, buf.String())
	}
}

func AddReplyInt(s *Server, c *Client, i int64) {
	if i == 0 {
		AddReply(s, c, s.Shared.Zero)
	} else if i == 1 {
		AddReply(s, c, s.Shared.One)
	} else {
		AddReplyIntWithPrifix(s, c, i, ':')
	}
}

func AddReplyMultiBulkLength(s *Server, c *Client, length int64) {
	AddReplyIntWithPrifix(s, c, length, '*')
}

/* Create the length prefix of a bulk reply, example: $2234 */
func AddReplyBulkLengthString(s *Server, c *Client, str string) {
	length := int64(len(str))
	AddReplyIntWithPrifix(s, c, length, '$')
}

func AddReplyBulkLengthStrObj(s *Server, c *Client, o *StrObject) {
	if !CheckRType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	if err == nil {
		AddReplyBulkLengthString(s, c, str)
	} else {
		return
	}
}

func AddReplyBulk(s *Server, c *Client, o *StrObject) {
	AddReplyBulkLengthStrObj(s, c, o)
	AddReplyStrObj(s, c, o)
	AddReply(s, c, s.Shared.Crlf)
}

func AddReplyBulkString(s *Server, c *Client, str string) {
	if str == "" {
		AddReply(s, c, s.Shared.NullBulk)
	} else {
		AddReplyBulkLengthString(s, c, str)
		AddReply(s, c, str)
		AddReply(s, c, s.Shared.Crlf)
	}
}

func AddReplyBulkInt(s *Server, c *Client, i int64) {
	str := strconv.FormatInt(i, 10)
	AddReplyBulkString(s, c, str)
}

func AddReplySubcommandSyntaxError(s *Server, c *Client) {
	cmd := c.Argv[0]
	AddReplyErrorFormat(s, c, "Unknown subcommand or wrong number of arguments for '%s'. Try %s HELP.", cmd, strings.ToUpper(cmd))
}
