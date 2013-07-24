package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

const (
	WIN = 100
)

func main() {

	rand.Seed(time.Now().Unix())

	l, err := net.Listen("tcp", "localhost:4000")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go Play(conn)
	}
}

func Play(conn net.Conn) {
	defer conn.Close()
again:
	conn.Write([]byte("Game Begin\n"))
	var players chan Player
	players = make(chan Player, 2)
	players <- &computer{"computer1", 0, 27}

	conn.Write([]byte("----------------------------\ninput your name plase:"))
	var buffer []byte
	buffer = make([]byte, 3)
	_, err := conn.Read(buffer)
	if err != nil {
		fmt.Println(err)
	}
	players <- &user{string(buffer), 0, 0, 0}

	for {
		player := <-players
		conn.Write([]byte(fmt.Sprintf("----------Now %s turn----------\n", player.GetName())))
		score := player.Play(conn)
		if score > WIN {
			conn.Write([]byte(fmt.Sprintf("GAME OVER.\n %s win.\n", player.GetName())))
			conn.Write([]byte(fmt.Sprintf("Play again? y/n\n")))

			var buffer []byte
			buffer = make([]byte, 2)
			_, err := conn.Read(buffer)
			if err != nil {
				fmt.Println(err)
				break
			}
			if string(buffer) == "n\n" {
				break
			} else {
				goto again
			}
		}
		players <- player
	}
}

type Player interface {
	Play(conn net.Conn) int
	GetName() string
}

type computer struct {
	Name  string
	Score int
	max   int
}

func (player *computer) GetName() string {
	return player.Name
}

func (player *computer) Play(conn net.Conn) int {
	thisTurn := 0
	for thisTurn < player.max {
		currentScore := rand.Intn(6) + 1
		conn.Write([]byte(fmt.Sprintf("This turn %d, rolling... get %d.\n", thisTurn, currentScore)))
		if currentScore == 1 {
			thisTurn = 0
			break
		} else {
			thisTurn += currentScore
		}
	}
	player.Score += thisTurn
	conn.Write([]byte(fmt.Sprintf("Finally, total score %d, this turn %d.\n", player.Score, thisTurn)))
	return player.Score
}

type CmdMessage struct {
	CMD uint32 //ROLL or STAY
}

const (
	ROLL = iota
	STAY
)

type user struct {
	Name      string
	Score     int
	ThisTurn  int
	RollScore int
}

func (player *user) GetName() string {
	return player.Name
}

func (player *user) Roll() (end bool) {
	end = false
	player.RollScore = rand.Intn(6) + 1
	if player.RollScore == 1 {
		player.ThisTurn = 0
		end = true
	} else {
		player.ThisTurn += player.RollScore
	}
	return
}
func (player *user) Play(conn net.Conn) int {
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	end := false
	player.ThisTurn = 0
	player.RollScore = 0
	for !end {
		enc.Encode(player)
		conn.Write([]byte(fmt.Sprintf("roll: {\"CMD\":0} or stay: {\"CMD\":1}?\n")))
		var msg CmdMessage
		if err := dec.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			conn.Write([]byte(fmt.Sprintf("%v", err)))
			log.Fatal("Error when decode", err)
		} else {
			switch msg.CMD {
			case ROLL:
				conn.Write([]byte(fmt.Sprintf("Rolling... \n")))
				end = player.Roll()
			case STAY:
				end = true
			default:
				conn.Write([]byte(fmt.Sprintf("Invalid input, roll: {\"CMD\":0} or stay: {\"CMD\":1}?")))
			}
		}
	}
	player.Score += player.ThisTurn
	enc.Encode(player)
	return player.Score
}
