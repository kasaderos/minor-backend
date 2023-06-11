package main

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

const MaxPlayers = 100
const GroupingThreshold = 0.5

type PlayerID int64
type GroupID int64

type GetGroupRequest struct {
	PlayerID `json:"player_id"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

type RegisterPlayerRequest struct {
	PlayerID `json:"player_id"`
}

type Map struct {
	Width  float64
	Height float64
}

type Player struct {
	PlayerID
	GroupID
	X, Y float64
}

type Game struct {
	mxG     sync.Mutex
	Players map[PlayerID]*Player
	Map

	mx             sync.Mutex
	playersCounter int64
}

func isNearby(x1, y1, x2, y2 float64) bool {
	return math.Abs(x1-x2) < GroupingThreshold && math.Abs(y1-y2) < GroupingThreshold
}

// todo assigning group id
func (g *Game) getGroupID(p *Player) GroupID {
	g.mxG.Lock()
	defer g.mxG.Unlock()

	for id, player := range g.Players {
		if id == p.PlayerID {
			continue
		}
		if isNearby(player.X, player.Y, p.X, p.Y) {
			p.GroupID = player.GroupID
			return p.GroupID
		}
	}
	return p.GroupID
}

func (g *Game) GetPlayer(id PlayerID) (*Player, error) {
	g.mxG.Lock()
	defer g.mxG.Unlock()

	p, ok := g.Players[id]
	if !ok {
		return nil, fmt.Errorf("unknown player")
	}
	return p, nil
}

func (g *Game) AddPlayer(p *Player) error {
	g.mxG.Lock()
	defer g.mxG.Unlock()

	if _, ok := g.Players[p.PlayerID]; ok {
		return fmt.Errorf("player exists")
	}
	p.GroupID = GroupID(p.PlayerID)
	g.Players[p.PlayerID] = p
	return nil
}

func (g *Game) genPlayerID() (PlayerID, error) {
	g.mx.Lock()
	defer g.mx.Unlock()
	if g.playersCounter == MaxPlayers {
		return -1, fmt.Errorf("max players reached")
	}
	id := PlayerID(g.playersCounter)
	g.playersCounter++
	return id, nil
}

type Handler struct {
	Game
}

func (h *Handler) InitPlayerHandler(c *gin.Context) {
	var req RegisterPlayerRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	id, err := h.Game.genPlayerID()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	player := &Player{
		PlayerID: id,
		X:        rand.Float64() * h.Map.Width,
		Y:        rand.Float64() * h.Map.Height,
	}

	err = h.Game.AddPlayer(player)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"X": player.X,
		"Y": player.Y,
	})
}

func (h *Handler) GroupHandler(c *gin.Context) {
	var req GetGroupRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}

	player, err := h.Game.GetPlayer(req.PlayerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"msg": err.Error()})
		return
	}
	groupID := h.getGroupID(player)

	c.JSON(http.StatusOK, gin.H{
		"group_id": groupID,
	})
}

func main() {
	r := gin.Default()
	h := Handler{
		Game: Game{
			Players: make(map[PlayerID]*Player),
			Map:     Map{},
		},
	}
	// Group handler returns group_id of player
	r.GET("/group", h.GroupHandler)
	// init player adds player to a new game
	// it take player_id and returns random coord (x, y)
	r.POST("/init/player", h.InitPlayerHandler)
	r.Run("localhost:8080")
}
