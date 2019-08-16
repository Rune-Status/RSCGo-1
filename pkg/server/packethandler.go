package server

import "bitbucket.org/zlacki/rscgo/pkg/server/packets"

// TODO: Maybe load this from some sort of persistent storage medium, e.g YAML/TOML/JSON file

var Handlers = make(map[byte]func(*Client, *packets.Packet))

//HandlePacket Finds the mapped handler function for the specified packet, and calls it with the specified parameters.
func (c *Client) HandlePacket(p *packets.Packet) {
	handler, ok := Handlers[p.Opcode]
	if !ok {
		LogInfo.Printf("Unhandled Packet: {opcode:%d; length:%d};\n", p.Opcode, len(p.Payload))
		return
	}
	handler(c, p)
}
