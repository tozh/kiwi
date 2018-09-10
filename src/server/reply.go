package server

import (
	. "redigo/src/constant"
	"strconv"
	"strings"
	"fmt"
	"bytes"
)

func (s *Server) AddReplyToBuffer(c *Client, str string) int64 {
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
		return C_OK
	}
	if c.Reply.ListLength() > 0 {
		return C_ERR
	}
	available := cap(c.Buf)
	if len(str) > available {
		return C_ERR
	}
	copy(c.Buf[c.BufPos:], str)
	c.BufPos += int64(len(str))
	return C_OK
}

func (s *Server) AddReplyToList(c *Client, str string) {
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
		return
	}
	if c.Reply.ListLength() == 0 {
		c.Reply.ListAddNodeTail(&str)
		c.ReplySize += int64(len(str))
	} else {
		ln := c.Reply.ListTail()
		tail := *ln.Value.(*string)
		if tail != "" && (len(tail) >= len(str) || len(tail)+len(str) < PROTO_REPLY_CHUNK_BYTES) {
			tail = CatString(tail, str)
			ln.Value = &tail
			c.ReplySize += int64(len(str))
		} else {
			c.Reply.ListAddNodeTail(&str)
			c.ReplySize += int64(len(str))
		}

	}
	//AsyncCloseClientOnOutputBufferLimitReached(s, c)
}

func (s *Server) AddReply(c *Client, str string) {
	if PrepareClientToWrite(c) != C_OK {
		return
	}
	if s.AddReplyToBuffer(c, str) != C_OK {
		s.AddReplyToList(c, str)
	}
}

func (s *Server) AddReplyStrObj(c *Client, o *StrObject) {
	if !CheckRType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	if err == nil {
		s.AddReply(c, str)
	} else {
		return
	}
}

func (s *Server) AddReplyError(c *Client, str string) {
	if len(str) != 0 || str[0] != '-' {
		s.AddReply(c, "-ERR ")
	}
	s.AddReply(c, str)
	s.AddReply(c, "\r\n")
}

func (s *Server) AddReplyErrorFormat(c *Client, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	s.AddReplyError(c, str)
}

func (s *Server) AddReplyStatus(c *Client, str string) {
	s.AddReply(c, "+")
	s.AddReply(c, str)
	s.AddReply(c, "\r\n")
}

func (s *Server) AddReplyStatusFormat(c *Client, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	s.AddReplyStatus(c, str)
}

//func (s *Server) AddReplyHelp(c *Client, help []string) {
//	cmd := c.Argv[0]
//	s.AddReplyStatusFormat(c, "%s <subcommand> arg arg ... arg. Subcommands are:", cmd)
//	for _, h := range help {
//		s.AddReplyStatus(c, h)
//	}
//}

func (s *Server) AddReplyIntWithPrifix(c *Client, i int64, prefix byte) {
	/* Things like $3\r\n or *2\r\n are emitted very often by the protocol
	so we have a few shared objects to use if the integer is small
	like it is most of the times. */
	if prefix == '*' && i >= 0 && i < SHARED_BULKHDR_LEN {
		s.AddReply(c, s.Shared.MultiBulkHDR[i])
	} else if prefix == '$' && i >= 0 && i < SHARED_BULKHDR_LEN {
		s.AddReply(c, s.Shared.MultiBulkHDR[i])
	} else {
		str := strconv.FormatInt(i, 10)
		buf := bytes.Buffer{}
		buf.WriteByte(prefix)
		buf.WriteString(str)
		buf.WriteByte('\r')
		buf.WriteByte('\n')
		s.AddReply(c, buf.String())
	}
}

func (s *Server) AddReplyInt(c *Client, i int64) {
	if i == 0 {
		s.AddReply(c, s.Shared.Zero)
	} else if i == 1 {
		s.AddReply(c, s.Shared.One)
	} else {
		s.AddReplyIntWithPrifix(c, i, ':')
	}
}

func (s *Server) AddReplyMultiBulkLength(c *Client, length int64) {
	s.AddReplyIntWithPrifix(c, length, '*')
}

/* Create the length prefix of a bulk reply, example: $2234 */
func (s *Server) AddReplyBulkLengthString(c *Client, str string) {
	length := int64(len(str))
	s.AddReplyIntWithPrifix(c, length, '$')
}

func (s *Server) AddReplyBulkLengthStrObj(c *Client, o *StrObject) {
	if !CheckRType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	if err == nil {
		s.AddReplyBulkLengthString(c, str)
	} else {
		return
	}
}

func (s *Server) AddReplyBulk(c *Client, o *StrObject) {
	s.AddReplyBulkLengthStrObj(c, o)
	s.AddReplyStrObj(c, o)
	s.AddReply(c, s.Shared.Crlf)
}

func (s *Server) AddReplyBulkString(c *Client, str string) {
	if str == "" {
		s.AddReply(c, s.Shared.NullBulk)
	} else {
		s.AddReplyBulkLengthString(c, str)
		s.AddReply(c, str)
		s.AddReply(c, s.Shared.Crlf)
	}
}

func (s *Server) AddReplyBulkInt(c *Client, i int64) {
	str := strconv.FormatInt(i, 10)
	s.AddReplyBulkString(c, str)
}

func (s *Server) AddReplySubcommandSyntaxError(c *Client) {
	cmd := c.Argv[0]
	s.AddReplyErrorFormat(c, "Unknown subcommand or wrong number of arguments for '%s'. Try %s HELP.", cmd, strings.ToUpper(cmd))
}
