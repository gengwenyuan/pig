package main

import (
	"fmt"
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
	players <- &user{string(buffer), 0}

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

type user struct {
	Name  string
	Score int
}

func (player *user) GetName() string {
	return player.Name
}

func (player *user) Play(conn net.Conn) int {

	thisTurn := 0
	var buffer []byte
	buffer = make([]byte, 2)
	for {
		conn.Write([]byte(fmt.Sprintf("roll(r) or stay(s)?\n")))
		buffer[0] = 0
		_, err := conn.Read(buffer)
		if err != nil {
			fmt.Println(err)
			break
		}
		switch {
		case string(buffer) == "r\n":
			currentScore := rand.Intn(6) + 1
			conn.Write([]byte(fmt.Sprintf("This turn %d, rolling... get %d.\n", thisTurn, currentScore)))
			if currentScore == 1 {
				thisTurn = 0
				goto end
			} else {
				thisTurn += currentScore
			}
		case string(buffer) == "s\n":
			goto end
		default:
			time.Sleep(time.Second)
			conn.Write([]byte(fmt.Sprintf("Invalid input, roll(r) or stay(s)?\n")))
			continue
		}
	}
end:
	player.Score += thisTurn
	conn.Write([]byte(fmt.Sprintf("Finally, total score %d, this turn %d.\n", player.Score, thisTurn)))
	return player.Score
}
