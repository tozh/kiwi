package server

import "fmt"

func createKiwiServerEvents() (events *Events) {
	events = &Events{
		NumLoops: kiwiS.numLoops,
	}
	events.Opened = func(c *Client) (out []byte, opts Options, action Action) {
		fmt.Println("Opened")
		return
	}
	events.Closed = func(c *Client, err error) (action Action) {
		CloseClient(c)
		fmt.Println("Closed")
		return
	}
	events.Data = func(c *Client, in []byte) (out []byte, action Action) {
		fmt.Println("Data---->", string(in))
		c.QueryCount++
		c.Reset(in)
		ProcessInput(c)
		out = c.OutBuf.Bytes()
		fmt.Println("Data---->", string(out))
		return
	}
	events.Written = func(c *Client, n int) (action Action) {
		fmt.Println("Written")

		c.SetLastInteraction()
		return
	}
	events.Shutdown = func() {
		fmt.Println("Shutdown Action...Funished")
		return
	}
	return
}
