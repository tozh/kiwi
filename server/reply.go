package server

import (
	"strconv"
	"fmt"
	"sync/atomic"
)

func AddReply(c *KiwiClient, str string) {
	if c.PrepareClientToWrite() != C_OK {
		return
	}
	n, err := c.OutBuf.WriteString(str)
	if err != nil {
		return
	}
	atomic.AddInt64(&kiwiS.StatNetOutputBytes, int64(n))
}

func AddReplyStrObj(c *KiwiClient, o *StrObject) {
	if !CheckOType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	if err == nil {
		AddReply(c, str)
	} else {
		return
	}
}

func AddReplyError(c *KiwiClient, str string) {
	if len(str) != 0 || str[0] != '-' {
		AddReply(c, "-ERR ")
	}
	AddReply(c, str)
	AddReply(c, "\r\n")
}

func AddReplyErrorFormat(c *KiwiClient, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	AddReplyError(c, str)
}

func AddReplyStatus(c *KiwiClient, str string) {
	AddReply(c, "+")
	AddReply(c, str)
	AddReply(c, "\r\n")
}

func (s *Server) AddReplyStatusFormat(c *KiwiClient, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	AddReplyStatus(c, str)
}

//func (kiwiS *Server) AddReplyHelp(c *KiwiClient, help []string) {
//	cmd := c.Argv[0]
//	kiwiS.AddReplyStatusFormat(c, "%kiwiS <subcommand> arg arg ... arg. Subcommands are:", cmd)
//	for _, h := range help {
//		kiwiS.AddReplyStatus(c, h)
//	}
//}

func AddReplyIntWithPrifix(c *KiwiClient, i int, prefix byte) {
	/* Things like $3\r\n or *2\r\n are emitted very often by the protocol
	so we have a few shared objects to use if the integer is small
	like it is most of the times. */
	if prefix == '*' && i >= 0 && i < SHARED_BULKHDR_LEN {
		AddReply(c, kiwiS.Shared.MultiBulkHDR[i])
	} else if prefix == '$' && i >= 0 && i < SHARED_BULKHDR_LEN {
		AddReply(c, kiwiS.Shared.BulkHDR[i])
	} else {
		str := strconv.Itoa(i)
		buf := Buffer{}
		buf.WriteByte(prefix)
		buf.WriteString(str)
		buf.WriteString("\r\n")
		AddReply(c, buf.String())
	}
}

func AddReplyInt(c *KiwiClient, i int) {
	if i == 0 {
		AddReply(c, kiwiS.Shared.Zero)
	} else if i == 1 {
		AddReply(c, kiwiS.Shared.One)
	} else {
		AddReplyIntWithPrifix(c, i, ':')
	}
}

func AddReplyMultiBulkLen(c *KiwiClient, length int) {
	AddReplyIntWithPrifix(c, length, '*')
}

/* Create the length prefix of a bulk reply, example: $2234 */
func AddReplyBulkLenOfStr(c *KiwiClient, str string) {
	length := len(str)
	// fmt.Println(">>>>>>>>>>>>>>>>", length)
	AddReplyIntWithPrifix(c, length, '$')
}

func AddReplyBulkStrObj(c *KiwiClient, o *StrObject) {
	if !CheckOType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	// fmt.Println(">>>>>>>>>>>>>>>>", str)
	if err == nil {
		AddReplyBulkLenOfStr(c, str)
	} else {
		return
	}
	AddReply(c, str)
	AddReply(c, kiwiS.Shared.Crlf)
}

func AddReplyBulkStr(c *KiwiClient, str string) {
	if str == "" {
		AddReply(c, kiwiS.Shared.NullBulk)
	} else {
		AddReplyBulkLenOfStr(c, str)
		AddReply(c, str)
		AddReply(c, kiwiS.Shared.Crlf)
	}
}

func AddReplyBulkInt(c *KiwiClient, i int) {
	str := strconv.Itoa(i)
	AddReplyBulkStr(c, str)
}

//
//func AddBuffer(buf *Buffer, str string) {
//	buf.WriteString(str)
//}
//
//func AddBufferByte(buf *Buffer, b byte) {
//	buf.WriteByte(b)
//}
//
//func AddBufferBytes(buf *Buffer, b []byte) {
//	buf.Write(b)
//}
//
//func AddBufferStrObj(buf *Buffer, o *StrObject) {
//	if !CheckOType(o, OBJ_RTYPE_STR) {
//		return
//	}
//	str, err := GetStrObjectValueString(o)
//	if err == nil {
//		AddBuffer(buf, str)
//	} else {
//		return
//	}
//}
//
//func AddBufferError(buf *Buffer, str string) {
//	if len(str) != 0 || str[0] != '-' {
//		AddBuffer(buf, "-ERR ")
//	}
//	AddBuffer(buf, str)
//	AddBuffer(buf, "\r\n")
//}
//
//func AddBufferErrorFormat(buf *Buffer, format string, a ...interface{}) {
//	str := fmt.Sprintf(format, a)
//	AddBufferError(buf, str)
//}
//
//func AddBufferStatus(buf *Buffer, str string) {
//	AddBuffer(buf, "+")
//	AddBuffer(buf, str)
//	AddBuffer(buf, "\r\n")
//}
//
//func (s *Server) AddBufferStatusFormat(buf *Buffer, format string, a ...interface{}) {
//	str := fmt.Sprintf(format, a)
//	AddBufferStatus(buf, str)
//}
//
////func (kiwiS *Server) AddReplyHelp(c *KiwiClient, help []string) {
////	cmd := c.Argv[0]
////	kiwiS.AddReplyStatusFormat(c, "%kiwiS <subcommand> arg arg ... arg. Subcommands are:", cmd)
////	for _, h := range help {
////		kiwiS.AddReplyStatus(c, h)
////	}
////}
//
//func AddBufferIntWithPrifix(buf *Buffer, i int, prefix byte) {
//	/* Things like $3\r\n or *2\r\n are emitted very often by the protocol
//	so we have a few shared objects to use if the integer is small
//	like it is most of the times. */
//	if prefix == '*' && i >= 0 && i < SHARED_BULKHDR_LEN {
//		AddBuffer(buf, kiwiS.Shared.MultiBulkHDR[i])
//	} else if prefix == '$' && i >= 0 && i < SHARED_BULKHDR_LEN {
//		AddBuffer(buf, kiwiS.Shared.BulkHDR[i])
//	} else {
//		AddBufferByte(buf, prefix)
//		AddBuffer(buf, strconv.Itoa(i))
//		AddBuffer(buf, "\r\n")
//	}
//}
//
//func AddBufferInt(buf *Buffer, i int) {
//	if i == 0 {
//		AddBuffer(buf, kiwiS.Shared.Zero)
//	} else if i == 1 {
//		AddBuffer(buf, kiwiS.Shared.One)
//	} else {
//		AddBufferIntWithPrifix(buf, i, ':')
//	}
//}
//
//func AddBufferMultiBulkLen(buf *Buffer, length int) {
//	AddBufferIntWithPrifix(buf, length, '*')
//}
//
///* Create the length prefix of a bulk reply, example: $2234 */
//func AddBufferBulkLenOfStr(buf *Buffer, str string) {
//	length := len(str)
//	// fmt.Println(">>>>>>>>>>>>>>>>", length)
//	AddBufferIntWithPrifix(buf, length, '$')
//}
//
//func AddBufferBulkStrObj(buf *Buffer, o *StrObject) {
//	if !CheckOType(o, OBJ_RTYPE_STR) {
//		return
//	}
//	str, err := GetStrObjectValueString(o)
//	// fmt.Println(">>>>>>>>>>>>>>>>", str)
//	if err == nil {
//		AddBufferBulkLenOfStr(buf, str)
//	} else {
//		return
//	}
//	AddBuffer(buf, str)
//	AddBuffer(buf, kiwiS.Shared.Crlf)
//}
//
//func AddBufferBulkStr(buf *Buffer, str string) {
//	if str == "" {
//		AddBuffer(buf, kiwiS.Shared.NullBulk)
//	} else {
//		AddBufferBulkLenOfStr(buf, str)
//		AddBuffer(buf, str)
//		AddBuffer(buf, kiwiS.Shared.Crlf)
//	}
//}
//
//func AddBufferBulkInt(buf *Buffer, i int) {
//	AddBufferBulkStr(buf, strconv.Itoa(i))
//}
