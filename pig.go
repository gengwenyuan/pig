package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
		enc := json.NewEncoder(conn)
		dec := json.NewDecoder(conn)
		var player user
		conn.Write([]byte("input your name plase: {\"Name\":\"gwy\"}\n"))
		if err := dec.Decode(&player); err == io.EOF {
			enc.Encode(fmt.Sprintf("err: %v", err))
		} else if err != nil {
			enc.Encode(fmt.Sprintf("err: %v", err))
		} else {
			player.Score = 0
			player.Rwc = conn
		}
		go match(&player)
	}
}

var players = make(chan Player)

func match(p Player) {
	p.Write([]byte("Waiting for a player..."))
	//fmt.Fprint(p "Waiting for a player...")

	select {
	case players <- p:
		// now handled by the other goroutine
	case p2 := <-players:
		Play(p, p2)
	case <-time.After(5 * time.Second):
		//r, w := io.Pipe()
		w := ioutil.Discard
		c := &computer{w, "computer1", 0, 0, 0, 27}
		Play(p, c)
	}
}

func Play(a, b Player) {
	defer a.Close()
	defer b.Close()

again:
	a.Reset()
	b.Reset()
	a.Write([]byte("Game Begin\n"))
	a.Write([]byte("----------------------------\n"))
	b.Write([]byte("Game Begin\n"))
	b.Write([]byte("----------------------------\n"))
	var players = make(chan Player, 2)
	players <- a
	players <- b
	for {
		player := <-players
		a.Write([]byte(fmt.Sprintf("----------Now %s turn----------\n", player.GetName())))
		b.Write([]byte(fmt.Sprintf("----------Now %s turn----------\n", player.GetName())))
		score := player.Play()
		if score > WIN {
			a.Write([]byte(fmt.Sprintf("GAME OVER.\n %s win.\n", player.GetName())))
			a.Write([]byte(fmt.Sprintf("Play again? y/n\n")))
			b.Write([]byte(fmt.Sprintf("GAME OVER.\n %s win.\n", player.GetName())))
			b.Write([]byte(fmt.Sprintf("Play again? y/n\n")))

			var buffer []byte
			buffer = make([]byte, 2)
			_, err := a.Read(buffer)
			if err != nil {
				fmt.Println(err)
				break
			}
			if string(buffer) == "n\n" {
				break
			} else {
				players <- player
				goto again //need modfiy here
			}
		}
		a.Write([]byte(fmt.Sprintf("Player %s get %d in this turn, now total score is %d\n",
			player.GetName(), player.GetThisTurn(), player.GetScore())))
		b.Write([]byte(fmt.Sprintf("Player %s get %d in this turn, now total score is %d\n",
			player.GetName(), player.GetThisTurn(), player.GetScore())))
		players <- player
	}

}

type Player interface {
	io.ReadWriteCloser
	Play() int
	GetName() string
	GetThisTurn() int
	GetScore() int
	Reset()
}

type computer struct {
	//Rc        io.ReadCloser
	Wt        io.Writer
	Name      string
	Score     int
	ThisTurn  int
	RollScore int
	max       int
}

func (player *computer) Read(p []byte) (n int, err error) {
	return 0, nil //player.Rc.Read(p)
}
func (player *computer) Write(p []byte) (n int, err error) {
	return player.Wt.Write(p)
}
func (player *computer) Close() error {
	return nil //player.Rc.Close()
}
func (player *computer) GetName() string {
	return player.Name
}
func (player *computer) GetThisTurn() int {
	return player.ThisTurn
}
func (player *computer) GetScore() int {
	return player.Score
}
func (player *computer) Reset() {
	player.Score = 0
	player.ThisTurn = 0
	player.RollScore = 0
	return
}
func (player *computer) Play() int {
	player.ThisTurn = 0
	for player.ThisTurn < player.max {
		player.RollScore = rand.Intn(6) + 1
		player.Wt.Write([]byte(fmt.Sprintf("This turn %d, rolling... get %d.\n", player.ThisTurn, player.RollScore)))
		if player.RollScore == 1 {
			player.ThisTurn = 0
			break
		} else {
			player.ThisTurn += player.RollScore
		}
	}
	player.Score += player.ThisTurn
	player.Wt.Write([]byte(fmt.Sprintf("Finally, total score %d, this turn %d.\n", player.Score, player.ThisTurn)))
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
	Rwc       io.ReadWriteCloser
	Name      string
	Score     int
	ThisTurn  int
	RollScore int
}

func (player *user) Read(p []byte) (n int, err error) {
	return player.Rwc.Read(p)
}
func (player *user) Write(p []byte) (n int, err error) {
	return player.Rwc.Write(p)
}
func (player *user) Close() error {
	return player.Rwc.Close()
}

func (player *user) GetName() string {
	return player.Name
}
func (player *user) GetThisTurn() int {
	return player.ThisTurn
}
func (player *user) GetScore() int {
	return player.Score
}
func (player *user) Reset() {
	player.Score = 0
	player.ThisTurn = 0
	player.RollScore = 0
	return
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
func (player *user) Play() int {
	dec := json.NewDecoder(player.Rwc)
	enc := json.NewEncoder(player.Rwc)
	end := false
	player.ThisTurn = 0
	player.RollScore = 0
	for !end {
		enc.Encode(player)
		player.Rwc.Write([]byte(fmt.Sprintf("roll: {\"CMD\":0} or stay: {\"CMD\":1}?\n")))
		var msg CmdMessage
		if err := dec.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			player.Rwc.Write([]byte(fmt.Sprintf("%v", err)))
			log.Fatal("Error when decode: ", err)
		} else {
			switch msg.CMD {
			case ROLL:
				player.Rwc.Write([]byte(fmt.Sprintf("Rolling... \n")))
				end = player.Roll()
			case STAY:
				end = true
			default:
				player.Rwc.Write([]byte(fmt.Sprintf("Invalid input, roll: {\"CMD\":0} or stay: {\"CMD\":1}?")))
			}
		}
	}
	player.Score += player.ThisTurn
	enc.Encode(player)
	return player.Score
}
