package server

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"bitbucket.org/zlacki/rscgo/pkg/entity"
	"bitbucket.org/zlacki/rscgo/pkg/server/errors"
	"bitbucket.org/zlacki/rscgo/pkg/server/packets"
)

//Client Represents a single connecting client.
type Client struct {
	isaacSeed        []uint64
	isaacStream      *IsaacSeed
	uID              uint8
	ip               string
	Index            int
	kill             chan struct{}
	awaitTermination sync.WaitGroup
	player           *entity.Player
	socket           net.Conn
	packetQueue      chan *packets.Packet
	outgoingPackets  chan *packets.Packet
	buffer           []byte
}

//StartReader Starts the clients socket reader goroutine.  Takes a waitgroup as an argument to facilitate synchronous destruction.
func (c *Client) StartReader() {
	defer c.awaitTermination.Done()
	for range time.Tick(50 * time.Millisecond) {
		select {
		default:
			p, err := c.ReadPacket()
			if err != nil {
				if err, ok := err.(errors.NetError); ok {
					if err.Closed || err.Ping {
						// TODO: I need to make sure this doesn't cause a panic due to kill being closed already
						return
					}
					LogError.Printf("Rejected Packet from: '%s'\n", c.ip)
					LogError.Println(err)
				}
				continue
			}
			c.packetQueue <- p
		case <-c.kill:
			return
		}
	}
}

//StartWriter Starts the clients socket writer goroutine.  Takes a waitgroup as an argument to facilitate synchronous destruction.
func (c *Client) StartWriter() {
	defer c.awaitTermination.Done()
	for range time.Tick(50 * time.Millisecond) {
		select {
		case p := <-c.outgoingPackets:
			if p == nil {
				return
			}
			c.WritePacket(p)
		case <-c.kill:
			return
		}
	}
}

//Destroy Safely tears down a client, saves it to the database, and removes it from server-wide collections.
func (c *Client) Destroy() {
	c.awaitTermination.Wait()
	entity.GetRegion(c.player.X(), c.player.Y()).RemovePlayer(c.player)
	c.player.Removing = true
	c.player.Connected = false
	close(c.outgoingPackets)
	close(c.packetQueue)
	if err := c.socket.Close(); err != nil {
		LogError.Println("Couldn't close socket:", err)
	}
	if c1, ok := Clients[c.player.UserBase37]; c1 == c && ok {
		c.Save()
		delete(Clients, c.player.UserBase37)
	}
	if ok := ClientList.Remove(c.Index); ok {
		LogInfo.Printf("Unregistered: %v\n", c)
	}
}

func (c *Client) ResetUpdateFlags() {
	c.player.Removing = false
	c.player.HasMoved = false
	c.player.AppearanceChanged = false
	c.player.HasSelf = true
}

func (c *Client) UpdatePositions() {
	if c.player.Location().Equals(entity.DeathSpot) || !c.player.Connected {
		return
	}
	var localPlayers []*entity.Player
	var localAppearances []*entity.Player
	var removingPlayers []*entity.Player
	var localObjects []*entity.Object
	var removingObjects []*entity.Object
	for _, r := range entity.SurroundingRegions(c.player.X(), c.player.Y()) {
		for _, p := range r.Players {
			if p.Index != c.Index {
				if c.player.Location().LongestDelta(p.Location()) <= 15 {
					if !c.player.LocalPlayers.ContainsPlayer(p) {
						localPlayers = append(localPlayers, p)
					}
				} else {
					if c.player.LocalPlayers.ContainsPlayer(p) {
						removingPlayers = append(removingPlayers, p)
					}
				}
			}
		}
		for _, o := range r.Objects {
			if c.player.Location().LongestDelta(o.Location()) <= 20 {
				if !c.player.LocalObjects.ContainsObject(o) {
					localObjects = append(localObjects, o)
				}
			} else {
				if c.player.LocalObjects.ContainsObject(o) {
					removingObjects = append(removingObjects, o)
				}
			}
		}
	}
	// TODO: Clean up appearance list code.
	for _, index := range c.player.Appearances {
		v := ClientList.Get(index)
		if v, ok := v.(*Client); ok {
			localAppearances = append(localAppearances, v.player)
		}
	}
	localAppearances = append(localAppearances, localPlayers...)
	c.player.Appearances = c.player.Appearances[:0]
	// POSITIONS BEFORE EVERYTHING ELSE.
	if positions := packets.PlayerPositions(c.player, localPlayers, removingPlayers); positions != nil {
		c.outgoingPackets <- positions
	}
	if appearances := packets.PlayerAppearances(c.player, localAppearances); appearances != nil {
		c.outgoingPackets <- appearances
	}
	if objectUpdates := packets.ObjectLocations(c.player, localObjects, removingObjects); objectUpdates != nil {
		c.outgoingPackets <- objectUpdates
	}
}

//StartNetworking Starts up 3 new goroutines; one for reading incoming data from the socket, one for writing outgoing data to the socket, and one for client state updates and parsing plus handling incoming packets.  When the clients kill signal is sent through the kill channel, the state update and packet handling goroutine will wait for both the reader and writer goroutines to complete their operations before unregistering the client.
func (c *Client) StartNetworking() {
	c.awaitTermination.Add(2)
	go c.StartReader()
	go c.StartWriter()
	go func() {
		defer c.Destroy()
		for {
			select {
			case p := <-c.packetQueue:
				if p == nil {
					return
				}
				c.HandlePacket(p)
			case <-c.kill:
				return
			}
		}
	}()
}

func (c *Client) sendLoginResponse(i byte) {
	c.outgoingPackets <- packets.LoginResponse(int(i))
	if i != 0 {
		LogInfo.Printf("Denied Client[%v]: {ip:'%v', username:'%v', Response='%v'}\n", c.Index, c.ip, c.player.Username, i)
		close(c.kill)
	} else {
		LogInfo.Printf("Registered Client[%v]: {ip:'%v', username:'%v'}\n", c.Index, c.ip, c.player.Username)
		entity.GetRegionFromLocation(c.player.Location()).AddPlayer(c.player)
		c.player.AppearanceChanged = true
		c.player.Connected = true
		for i := 0; i < 18; i++ {
			level := 1
			exp := 0
			if i == 3 {
				level = 10
				exp = 1154
			}
			c.player.Skillset.Current[i] = level
			c.player.Skillset.Maximum[i] = level
			c.player.Skillset.Experience[i] = exp
		}
		c.outgoingPackets <- packets.PlayerInfo(c.player)
		c.outgoingPackets <- packets.PlayerStats(c.player)
		c.outgoingPackets <- packets.EquipmentStats(c.player)
		c.outgoingPackets <- packets.FightMode(c.player)
		c.outgoingPackets <- packets.FriendList(c.player)
		c.outgoingPackets <- packets.ClientSettings(c.player)
		c.outgoingPackets <- packets.Fatigue(c.player)
		c.outgoingPackets <- packets.WelcomeMessage
		c.outgoingPackets <- packets.ServerInfo(len(Clients))
		c.outgoingPackets <- packets.LoginBox(0, c.ip)
	}
}

//NewClient Creates a new instance of a Client, launches goroutines to handle I/O for it, and returns a reference to it.
func NewClient(socket net.Conn) *Client {
	c := &Client{socket: socket, isaacSeed: make([]uint64, 2), packetQueue: make(chan *packets.Packet, 25), ip: strings.Split(socket.RemoteAddr().String(), ":")[0], Index: -1, kill: make(chan struct{}), player: entity.NewPlayer(), buffer: make([]byte, 5000), outgoingPackets: make(chan *packets.Packet, 25)}
	c.StartNetworking()
	return c
}

//String Returns a string populated with some of the more identifying fields from the receiver Client.
func (c *Client) String() string {
	return fmt.Sprintf("Client[%v] {username:'%v', ip:'%v'}", c.Index, c.player.Username, c.ip)
}
