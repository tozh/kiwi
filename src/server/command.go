package server

import (
	."redigo/src/object"
	."redigo/src/constant"
	"strings"
)

/* SET key value [NX] [XX] [EX <seconds>] [PX <milliseconds>] */
// NX - not exist
// XX - exist
// EX - expire in seconds
// PX - expire in milliseconds
func setCommand(c *Client) {
	flags := OBJ_SET_NO_FLAGS
	for j:=int64(3); j<c.Argc;j++ {
		a := strings.ToUpper(*c.Argv[j].(*StrObject).Value)

		if a == "NX" && (flags & OBJ_SET_XX) != 0 {
			flags |= OBJ_SET_NX
		} else if a == "XX" && (flags & OBJ_SET_NX) != 0 {
			flags |= OBJ_SET_XX
		} else {
			// addReply
			return
		}
		// expire is not implemented now, so end here
	}

	//c.Argv[2] = tryObjectEncoding(c.Argv[2])
	setGenericCommand(c, int64(flags), c.Argv[1], c.Argv[2])
}

func setGenericCommand(c *Client, flags int64, key IObject, value IObject) {

}
