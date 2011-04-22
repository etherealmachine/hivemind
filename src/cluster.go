package main

import (
	"redis"
	"bytes"
	"gob"
	"os"
	"fmt"
	"time"
	"strings"
)

// Master:
// 	 Publish to scatter/gather queue
// 	 Push to farm queue
// 	 Subscribe to response queue
// Worker:
//   Subscribe to scatter/gather queue
//   Pop from farm queue
//   Publish to response queue

var master bool
var client redis.Client
// response queue
var responses chan redis.Message
var registered bool

func SetupCluster() {
	if *server != "" {
		client.Addr = *server + ":6379"
	}

	if *server != "" && (*modeGTP || *test) {
		master = true
		responses = subscribe("responses")
	} else if *server != "" && !*train {
		master = false
		sg := make(chan string, 0)
		farm := make(chan string, 0)
		go func() {
			sgqueue := subscribe("sg")
			for {
				sg <- ToString((<-sgqueue).Message)
			}
		}()
		go func() {
			for {
				_, buf, _ := client.Blpop([]string{"farm"}, 0)
				farm <- ToString(buf)
			}
		}()
		for {
			select {
			case s := <-sg:
				p := strings.Split(s, ":", -1)
				t, id := p[0], p[1]
				HandleJob(t, id)
			case s := <-farm:
				p := strings.Split(s, ":", -1)
				t, id := p[0], p[1]
				HandleJob(t, id)
			}
		}
	}
}

func subscribe(queue string) chan redis.Message {
	ch := make(chan redis.Message, 0)
	sub := make(chan string, 1)
	sub <- queue
	go func() {
		err := client.Subscribe(sub, nil, nil, nil, ch)
		if err != nil {
			panic(err)
		}
	}()
	return ch
}

func ToString(buf []byte) (s string) {
	b := bytes.NewBuffer(buf)
	d := gob.NewDecoder(b)
	d.Decode(&s)
	return
}

func ToBytes(v ...interface{}) []byte {
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	for i := 0; i < len(v); i++ {
		if v[i] != nil {
			e.Encode(v[i])
		}
	}
	return b.Bytes()
}

func FromBytes(buf []byte, v ...interface{}) {
	if buf == nil {
		return
	}
	b := bytes.NewBuffer(buf)
	d := gob.NewDecoder(b)
	for i := 0; i < len(v); i++ {
		d.Decode(v[i])
	}
}

func uuid() string {
	f, _ := os.Open("/dev/urandom")
	b := make([]byte, 16)
	f.Read(b)
	f.Close()
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func JobBytes(t string, id string, resp string) []byte {
	return ToBytes(t + ":" + id + ":" + resp)
}

func HandleJob(t string, id string) {
	respid := uuid()
	switch t {
	case "ping":
		// do nothing, just publish response
	case "scatter":
		buf, err := client.Get(id)
		if err != nil {
			panic(err)
		}
		opts := &Options{}
		b := bytes.NewBuffer(buf)
		d := gob.NewDecoder(b)
		d.Decode(opts)
		opts.unwrap()
		var color byte
		t := NewTracker(*size)
		d.Decode(&color)
		d.Decode(t)
		root := NewRoot(color, t)
		genmove(root, t, nil)
		for child := root.child; child != nil; child = child.sibling {
			child.parent = nil
			child.child = nil
			child.last = nil
		}
		err = client.Set(respid, ToBytes(root))
		if err != nil {
			panic(err)
		}
	}
	err := client.Publish("responses", JobBytes("response", id, respid))
	if err != nil {
		panic(err)
	}
}

func Scatter(color byte, t Tracker) (int, string) {
	if !master {
		return 0, ""
	}
	id := uuid()
	err := client.Publish("sg", JobBytes("ping", id, ""))
	if err != nil {
		panic(err)
	}
	// gather the number of workers
	workers := 0
	for s := time.Nanoseconds(); (time.Nanoseconds() - s) < 3e9; {
		select {
		case <-responses:
			workers++
		default:
		}
		time.Sleep(1e8)
	}
	id = uuid()
	err = client.Set(id, ToBytes(NewOptions(), color, t))
	if err != nil {
		panic(err)
	}
	log.Println(id)
	err = client.Publish("sg", JobBytes("scatter", id, ""))
	if err != nil {
		panic(err)
	}
	return workers, id
}

func Gather(workers int, jid string, root *Node) {
	if !master {
		return
	}
	for ; workers > 0; workers-- {
		log.Printf("waiting on %d workers", workers)
		for {
			msg := <-responses
			p := strings.Split(ToString(msg.Message), ":", -1)
			_, id, respid := p[0], p[1], p[2]
			if id == jid {
				log.Println("got", respid)
				buf, err := client.Get(respid)
				if err != nil {
					panic(err)
				}
				_, err = client.Del(respid)
				if err != nil {
					panic(err)
				}
				n := &Node{}
				FromBytes(buf, n)
				root.merge(n)
				break
			} else {
				log.Println("ignoring", id)
			}
		}
	}
	_, err := client.Del(jid)
	if err != nil {
		panic(err)
	}
}

/*
 Return the result of playing one game
*/
func Farm(black PatternMatcher, white PatternMatcher) (t Tracker) {
	t = NewTracker(*size)
	passes := 0
	var vertex int
	for {
		br := NewRoot(BLACK, t)
		genmove(br, t, black)
		if br == nil || br.Best() == nil {
			vertex = -1
			passes++
		} else {
			passes = 0
			vertex = br.Best().vertex
		}
		t.Play(BLACK, vertex)
		if (*hex && t.Winner() != EMPTY) || passes == 2 {
			break
		}
		wr := NewRoot(WHITE, t)
		genmove(wr, t, black)
		if wr == nil || wr.Best() == nil {
			vertex = -1
			passes++
		} else {
			passes = 0
			vertex = wr.Best().vertex
		}
		t.Play(WHITE, vertex)
		if (*hex && t.Winner() != EMPTY) || passes == 2 {
			break
		}
	}
	log.Println(Bwboard(t.Board(), t.Boardsize(), true))
	return
}

type Options struct {
	maxPlayouts uint
	stats       bool
	hex         bool
	ttt         bool
	uct         bool
	pat         bool
	hand        bool
	c           float64
	k           float64
	expandAfter float64
}

func NewOptions() *Options {
	return &Options{*maxPlayouts, *stats, *hex, *ttt, *uct, *pat, *hand, *c, *k, *expandAfter}
}

func (opts *Options) unwrap() {
	*maxPlayouts = opts.maxPlayouts
	*stats = opts.stats
	*hex = opts.hex
	*ttt = opts.ttt
	*uct = opts.uct
	*pat = opts.pat
	*hand = opts.hand
	*c = opts.c
	*k = opts.k
	*expandAfter = opts.expandAfter
	if !registered {
		gob.Register(NewTracker(9))
		registered = true
	}
}
