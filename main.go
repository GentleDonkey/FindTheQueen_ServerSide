package main

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type candidate struct {
	Connection net.Conn
	Dealer     bool
	Result     []bool
}

var anthConbo = map[string]string{
	"dannyboi":"dre@margh_shelled",
	"matty7":"win&win99",
}

var connMap = &sync.Map{}

func main() {
	fmt.Println("Server started...")
	ln, err := net.Listen("tcp", ":7621")
	if err != nil {
		fmt.Println("Error starting socket server: " + err.Error())
	}

	var wg = sync.WaitGroup{}
	wg.Add(2)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println("Error listening to client: " + err.Error())
				continue
			}
			fmt.Println(conn.RemoteAddr().String() + ": client connected")
			fmt.Println(conn.RemoteAddr().String() + ": wait for auth")
			go authentication(conn, &wg)
		}
	}()
	wg.Wait()
	if lenSyncMap(connMap) >= 2 {
		fmt.Println("Two candidates joined, game starts")
		for i := 1; i < 6; i++ {
			roundStart(connMap, i)
		}
		gameOver(connMap)
	}
	time.Sleep(time.Duration(5) * time.Second)
}

func authentication(conn net.Conn, wg *sync.WaitGroup) bool {
	user, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println(conn.RemoteAddr().String() + ": client disconnected")
		conn.Close()
		fmt.Println(conn.RemoteAddr().String() + ": end receiving data")
		return false
	}
	user = strings.Trim(user, "\n")
	pwd, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		fmt.Println(conn.RemoteAddr().String() + ": client disconnected")
		conn.Close()
		fmt.Println(conn.RemoteAddr().String() + ": end receiving data")
		return false
	}
	pwd = strings.Trim(pwd, "\n")
	if _, ok := anthConbo[user]; ok {
		if anthConbo[user] == pwd {
			fmt.Println(conn.RemoteAddr().String() + ": Successfully login")
			id := uuid.New().String()
			connMap.Store(id, &candidate{conn, false, make([]bool, 5)})
			wg.Done()
			return true
		}else{
			fmt.Println("Failed login")
			if _, err := fmt.Fprintf(conn, "Wrong credential. Please exit and restart this program.\n") ; err != nil {
				fmt.Println("Error listening to client: " + err.Error())
			}
			conn.Close()
		}
	}else{
		fmt.Println("Failed login")
		if _, err := fmt.Fprintf(conn, "Wrong credential. Please exit and restart this program.\n") ; err != nil {
			fmt.Println("Error listening to client: " + err.Error())
		}
		conn.Close()
	}
	return false
}

func roundStart(connMap *sync.Map, round int) {
	spotterMsg := "Round "+strconv.Itoa(round)+" started, you are the spotter, please guess the number [1,2,3]\n"
	dealerMsg := "Round "+strconv.Itoa(round)+" started, you are the dealer, please set the number [1,2,3]\n"
	firstUserIsHost := randomBool()
	i := 0
	wg := sync.WaitGroup{}
	var inputs []string
	wg.Add(2)
	fmt.Println("Round", round, "began")
	connMap.Range(func(key, value interface{}) bool {
		c, ok := value.(*candidate)
		if !ok {
			fmt.Println("error praising value to interface")
		}
		var message string
		if round == 1 {
			if firstUserIsHost {
				if i == 0{
					c.Dealer = true
					message = dealerMsg
				} else{
					c.Dealer = false
					message = spotterMsg
				}
			}
			if !firstUserIsHost{
				if i == 0{
					c.Dealer = false
					message = spotterMsg
				} else{
					c.Dealer = true
					message = dealerMsg
				}
			}
		} else {
			c.Dealer = !c.Dealer
			if c.Dealer {
				message = dealerMsg
			} else {
				message = spotterMsg
			}
		}
		if _, err := fmt.Fprintf(c.Connection, message) ; err != nil {
			fmt.Println("Error listening to client: " + err.Error())
		}
		i++
		go func() {
			input, err := bufio.NewReader(c.Connection).ReadString('\n')
			input = strings.Trim(input, "\n")
			if err != nil {
				fmt.Errorf("error at reading number %v", err)
				c.Connection.Close()
			}
			wg.Done()
			inputs = append(inputs, input)
		}()
		return true
	})
	wg.Wait()
	connMap.Range(func(key, value interface{}) bool {
		winMsg := "In round "+strconv.Itoa(round)+", you won\n"
		loseMsg := "In round "+strconv.Itoa(round)+", you lost\n"
		var message string
		c, ok := value.(*candidate)
		//fmt.Sprintf("compare number dealer: %v", c.Dealer)
		if !ok {
			fmt.Println("error praising value to interface")
		}
		if c.Dealer {
			if inputs[0] == inputs[1] {
				c.Result[round-1] = false
				message = loseMsg
			} else {
				c.Result[round-1] = true
				message = winMsg
			}
		} else {
			if inputs[0] == inputs[1] {
				c.Result[round-1] = true
				message = winMsg
			} else {
				c.Result[round-1] = false
				message = loseMsg
			}
		}
		if _, err := fmt.Fprintf(c.Connection, message) ; err != nil {
			fmt.Println("Error listening to client: " + err.Error())
		}
		return true
	})
	fmt.Println("Round", round, "completed")
}

func gameOver(connMap *sync.Map) {
	fmt.Println("Calculating result...")
	connMap.Range(func(key, value interface{}) bool {
		c, ok := value.(*candidate)
		if !ok {
			fmt.Println("Error praising value to interface")
		}
		var winNum = 0
		for _, v := range c.Result {
			if v == true{
				winNum++
			}
		}
		if winNum >= 3 {
			fmt.Println(c.Connection.RemoteAddr().String(), "wins")
			message := "Victory. Thanks for playing.\n"
			if _, err := fmt.Fprintf(c.Connection, message) ; err != nil {
				fmt.Println("Error listening to client: " + err.Error())
			}
		}else{
			fmt.Println(c.Connection.RemoteAddr().String(), "loses")
			message := "Defeat. Thanks for playing.\n"
			if _, err := fmt.Fprintf(c.Connection, message) ; err != nil {
				fmt.Println("Error listening to client: " + err.Error())
			}
		}
		return true
	})
	fmt.Println("Game over")
}

//return length of syn map
func lenSyncMap(m *sync.Map) int {
	var i int
	m.Range(func(k, v interface{}) bool {
		i++
		return true
	})
	return i
}

//return a randomly boolean
func randomBool() bool {
	return rand.Int()%2 == 0
}