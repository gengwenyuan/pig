package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"time"
)

const (
	tcpListenAddr = "localhost:4000"
	webListenAddr = "localhost:4001"
	WIN           = 100
)

func main() {
	rand.Seed(time.Now().Unix())

	//tcp server
	go netListen()

	//http server
	http.HandleFunc("/", rootHandler)
	http.Handle("/pig", websocket.Handler(websocketHandler))
	err := http.ListenAndServe(webListenAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func netListen() {
	l, err := net.Listen("tcp", tcpListenAddr)
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

func rootHandler(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("pig.html")
	t.Execute(w, webListenAddr)
}

type webplayer struct {
	Player
	done chan bool
}

func (s webplayer) Close() error {
	s.done <- true
	return nil
}

func websocketHandler(ws *websocket.Conn) {
	u := user{Rwc: ws, Name: "user1"}
	s := webplayer{&u, make(chan bool)}
	go match(&s)
	<-s.done
}

type Player interface {
	io.ReadWriteCloser
	Play(opponent Player) int
	GetName() string
	GetThisTurn() int
	GetScore() int
	Reset()
}

var players = make(chan Player)

func match(p Player) {
	p.Write([]byte("Waiting for a player...\n"))

	select {
	case players <- p:
		// now handled by the other goroutine
	case p2 := <-players:
		Play(p, p2)
	case <-time.After(5 * time.Second):
		//w := ioutil.Discard
		c := &computer{Name: "computer1", max: 27}
		Play(p, c)
	}
}

func Play(a, b Player) {
	defer a.Close()
	defer b.Close()

again:
	a.Reset()
	b.Reset()
	both := io.MultiWriter(a, b)
	both.Write([]byte(fmt.Sprintf("%s VS %s, Game Begin\n", a.GetName(), b.GetName())))
	both.Write([]byte("----------------------------\n"))
	current, opponent := a, b
	for {
		score := current.Play(opponent)
		if score > WIN {
			both.Write([]byte(fmt.Sprintf("GAME OVER.\n %s win.\n", current.GetName())))
			both.Write([]byte(fmt.Sprintf("Play again? y/n\n")))

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
				goto again //need modfiy here
			}
		}
		current, opponent = opponent, current
	}
}

type computer struct {
	//Rc        io.ReadCloser
	//Wt        io.Writer
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
	return 9, nil //player.Wt.Write(p)
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
func (player *computer) Play(opponent Player) int {
	opponent.Write([]byte(fmt.Sprintf("----------Now %s turn----------\n", player.GetName())))
	player.ThisTurn = 0
	end := false
	for player.ThisTurn < player.max && !end {
		player.RollScore = rand.Intn(6) + 1
		if player.RollScore == 1 {
			player.ThisTurn = 0
			time.Sleep(100 * time.Millisecond)
			end = true
		} else {
			player.ThisTurn += player.RollScore
		}
		opponent.Write([]byte(fmt.Sprintf("Rolling... get %d, this turn %d.\n",
			player.RollScore, player.ThisTurn)))
	}
	player.Score += player.ThisTurn
	opponent.Write([]byte(fmt.Sprintf("Finally, this turn %d, total score (%d vs %d).\n",
		player.ThisTurn, player.Score, opponent.GetScore())))
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
		time.Sleep(100 * time.Millisecond)
	}
	return
}
func (player *user) Play(opponent Player) int {
	both := io.MultiWriter(player, opponent)
	both.Write([]byte(fmt.Sprintf("----------Now %s turn----------\n", player.GetName())))
	dec := json.NewDecoder(player)
	//enc := json.NewEncoder(player)
	player.ThisTurn = 0
	player.RollScore = 0
	end := false
	for !end {
		//enc.Encode(player)
		player.Write([]byte(fmt.Sprintf("roll: {\"CMD\":0} or stay: {\"CMD\":1}?\n")))
		var msg CmdMessage
		if err := dec.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			player.Write([]byte(fmt.Sprintf("%v", err)))
			log.Fatal("Error when decode: ", err)
		} else {
			switch msg.CMD {
			case ROLL:
				end = player.Roll()
				both.Write([]byte(fmt.Sprintf("Rolling... get %d, this turn %d.\n",
					player.RollScore, player.ThisTurn)))
			case STAY:
				end = true
			default:
				player.Write([]byte(fmt.Sprintf("Invalid input, roll: {\"CMD\":0} or stay: {\"CMD\":1}?")))
			}
		}
	}
	player.Score += player.ThisTurn
	both.Write([]byte(fmt.Sprintf("Finally, this turn %d, total score (%d vs %d).\n",
		player.ThisTurn, player.Score, opponent.GetScore())))
	return player.Score
}
