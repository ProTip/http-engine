// checkengine project main.go
package main

import (
	"bytes"
	"dripper"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"
)

type HttpCheck struct {
	Url      string
	Interval int // How often we should check
	Timeout  int // Timeout in seconds applied as a hard IO deadline
}

type CheckStatus struct {
	Message      string
	ResponseTime time.Duration
}

var wg sync.WaitGroup
var timeout = time.Duration(6 * time.Second)

func main() {
	runtime.GOMAXPROCS(1)

	var codes = make(chan *CheckStatus, 50)
	d := dripper.NewDripper()
	urlsBytes, err := ioutil.ReadFile("urls.txt")
	_ = err
	urlsBytes = urlsBytes[:len(urlsBytes)-1]
	urls := bytes.Split(urlsBytes, []byte("\n"))
	for _, url := range urls {
		d.AddDrop(string(url), 0)
	}
	d.Drip()
	wg.Add(1)
	go collectDrops(d.Faucet, codes)
	go printStatus(codes)
	wg.Wait()
}

func printStatus(c chan *CheckStatus) {
	for {
		fmt.Println(<-c)
	}
}

func collectDrops(faucet chan interface{}, codes chan *CheckStatus) {
	for {
		key := <-faucet
		check := &HttpCheck{
			Url: key.(string),
		}
		go checkStatus(check, codes)
	}
}

func checkStatus(check *HttpCheck, codes chan *CheckStatus) {
	if check.Timeout == 0 {
		check.Timeout = 10
	}
	timeout := time.Duration(check.Timeout) * time.Second

	transport := http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			con, err := net.DialTimeout(network, addr, timeout)
			if err != nil {
				return nil, err
			}
			deadline := time.Now().Add(timeout)
			con.SetDeadline(deadline)
			return con, err
		},
	}
	client := http.Client{
		Transport: &transport,
	}

	start := time.Now()
	resp, err := client.Head(check.Url)
	if err == nil {
		end := time.Now()
		responseTime := end.Sub(start)
		httpStatus := &CheckStatus{resp.Status, responseTime}
		codes <- httpStatus
		defer resp.Body.Close()
	} else {
		end := time.Now()
		responseTime := end.Sub(start)
		httpStatus := &CheckStatus{err.Error(), responseTime}
		codes <- httpStatus
	}

	_ = err
	_ = resp
}
