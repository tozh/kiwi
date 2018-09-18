package server

import (
	"fmt"
	"sync/atomic"
	"strings"
	"strconv"
)

func CreateKiwiServerEvents() (events Events) {
	events = Events{
		NumLoops: kiwiS.numLoops,
	}
	events.Accepted = func(conn *conn, flags int) (c Client, action Action) {
		return CreateClient(conn, flags)
	}

	events.Opened = func(c Client) (out []byte, opts Options, action Action) {
		// fmt.Println("Opened")
		if kiwiS.StatConnCount > kiwiS.MaxClients {
			out = append([]byte{}, "-ERROR exceeds the maximum number of clients.\r\n"...)
			action = Close
		}
		return
	}
	events.Closed = func(c Client, err error) (action Action) {
		cli := c.(*KiwiClient)
		CloseClient(cli)
		// fmt.Println("Closed")
		return
	}
	events.Data = func(c Client, in []byte) (out []byte, action Action) {
		cli := c.(*KiwiClient)
		// fmt.Println("Data---->", string(in))
		if len(in) > 0 {
			atomic.AddInt64(&kiwiS.StatNetInputBytes, int64(len(in)))
		}
		cli.QueryCount++
		cli.Reset(in)
		ProcessInput(cli)
		cli.OutBuf.WriteByte(0)
		out = cli.OutBuf.Bytes()
		// fmt.Println("Data---->", string(out))
		return
	}
	events.Written = func(c Client, n int) (action Action) {
		cli := c.(*KiwiClient)
		// fmt.Println("Written")
		atomic.AddInt64(&kiwiS.StatNetOutputBytes, int64(n))
		cli.SetLastInteraction()
		return
	}
	events.Shutdown = func() {
		// fmt.Println("Shutdown Action...Funished")
		return
	}
	return
}

func Call(c *KiwiClient) {
	// fmt.Println("Call")
	c.Cmd.Process(c)
	atomic.AddInt64(&kiwiS.StatNumCommands, 1)
}

func ProcessCommand(c *KiwiClient) int {
	// fmt.Println("ProcessCommand")
	cmdName := strings.ToLower(c.Argv[0])
	// fmt.Println([]byte(cmdName))
	c.Cmd = LookUpCommand(cmdName)
	if c.Cmd == nil {
		// fmt.Println("c.Cmd == nil")
		AddReplyError(c, fmt.Sprintf("unknown command '%kiwiS'", cmdName))
		return C_OK
	}
	if (c.Cmd.Arity > 0 && c.Cmd.Arity != c.Argc) || c.Argc < -c.Cmd.Arity {
		AddReplyError(c, fmt.Sprintf("wrong number of arguments for '%kiwiS' command", cmdName))
		return C_OK
	}
	if kiwiS.RequirePassword != nil && c.Authenticated == 0 && &c.Cmd.Process != &AuthCommand {
		// fmt.Println("Authenticated")
		AddReplyError(c, kiwiS.Shared.NoAuthErr)
		return C_OK
	}
	Call(c)
	return C_OK
}

func LookUpCommand(name string) *Command {
	return kiwiS.Commands[name]
}

func ProcessInline(c *KiwiClient) int {
	// fmt.Println("ProcessInline")

	// Search for end of line
	queryBuf := c.InBuf.Bytes()
	size := len(queryBuf)
	newline := IndexOfBytes(queryBuf, 0, size, '\n')
	if newline == -1 {
		if size > kiwiS.ClientMaxQueryBufLen {
			AddReplyError(c, "Protocol error: too big inline request")
			//SetProtocolError(c, "too big inline request", 0)
		}
		return C_ERR
	}
	if newline != 0 && newline != size && queryBuf[newline-1] == '\r' {
		// Handle the \r\n case.
		newline--
	}
	/* Split the input buffer up to the \r\n */
	argvs := SplitArgs(queryBuf[0:newline])
	if argvs == nil {
		AddReplyError(c, "Protocol error: unbalanced quotes in request")
		//SetProtocolError(c, "unbalanced quotes in inline request", 0)
		return C_ERR
	}

	// Leave data after the first line of the query in the buffer
	if len(argvs) != 0 {
		c.Argc = 0
		c.Argv = make([]string, len(argvs))
	}
	for index, argv := range argvs {
		if argv != "" {
			c.Argv[index] = argv
			c.Argc++
		}
	}
	return C_OK
}

func ProcessMultiBulk(c *KiwiClient) int {
	if c.Argc != 0 {
		panic("c.Argc != 0")
	}
	if c.MultiBulkLen == 0 {
		star, err := c.InBuf.ReadByte()
		if err != nil || star != '*' {
			AddReplyError(c, fmt.Sprintf("Protocol error: expected '*', got '%c'", star))
			//SetProtocolError(c, "expected $ but got something else", 0)
			return C_ERR
		}
		bulkNumStr, err := c.InBuf.ReadStringExclude('\r')
		if err != nil {
			return C_ERR
		}

		bulkNum, err := strconv.Atoi(bulkNumStr)
		if err != nil || bulkNum > 1024*1024 {
			AddReplyError(c, "Protocol error: invalid multibulk length")
			//SetProtocolError(c, "invalid multibulk length", 0)
			return C_ERR
		}
		if bulkNum <= 0 {
			return C_OK
		}
		c.InBuf.ReadByte() // pass the \n
		c.MultiBulkLen = bulkNum
		c.Argv = make([]string, c.MultiBulkLen)
	}
	if c.MultiBulkLen < 0 {
		return C_ERR
	}
	for c.MultiBulkLen > 0 {
		// Read bulk length if unknown
		dollar, err := c.InBuf.ReadByte()
		if err != nil || dollar != '$' {
			AddReplyError(c, fmt.Sprintf("Protocol error: expected '$', got '%c'", dollar))
			return C_ERR
		}
		bulkLenStr, err := c.InBuf.ReadStringExclude('\r')
		if err != nil {
			AddReplyError(c, fmt.Sprintf("Protocol error: invalid bulk length"))
			return C_ERR
		}
		bulkLen, err := strconv.Atoi(bulkLenStr)
		if err != nil || bulkLen > kiwiS.ProtoMaxBulkLen {
			AddReplyError(c, "Protocol error: invalid bulk length")
			return C_ERR
		}
		c.InBuf.ReadByte() // pass the \n

		bulk := c.InBuf.Next(bulkLen)
		if len(bulk) != bulkLen {
			AddReplyError(c, "Protocol error: invalid bulk format")
			return C_ERR
		}
		cr, _ := c.InBuf.ReadByte()
		lf, _ := c.InBuf.ReadByte()
		if cr != '\r' || lf != '\n' {
			AddReplyError(c, "Protocol error: invalid bulk format")
			return C_ERR
		}
		c.Argv[len(c.Argv)-c.MultiBulkLen] = string(bulk)
		c.Argc++
		c.MultiBulkLen--
	}
	if c.MultiBulkLen == 0 {
		return C_OK
	}
	return C_ERR
}

func ProcessInput(c *KiwiClient) {
	if c.RequestType == 0 {
		firstByte, _ := c.InBuf.ReadByteNotGoForward()
		if firstByte == '*' {
			c.RequestType = PROTO_REQ_MULTIBULK
		} else {
			c.RequestType = PROTO_REQ_INLINE
		}
	}
	if c.RequestType == PROTO_REQ_INLINE {
		if ProcessInline(c) != C_OK {
		}
	} else if c.RequestType == PROTO_REQ_MULTIBULK {
		if ProcessMultiBulk(c) != C_OK {

		}
	} else {
		panic("Unknown request type")
	}

	if c.Argc != 0 {
		ProcessCommand(c)
	}
}
