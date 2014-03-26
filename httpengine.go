// checkengine project main.go
package httpengine

import (
	"fmt"
	"github.com/amir/raidman"
	"github.com/protip/dripper"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type HttpCheck struct {
	Address  string
	Host     string
	Path     string
	Interval int // How often we should check
	Timeout  int // Timeout in seconds applied as a hard IO deadline
}

type HttpEngine struct {
	dripper *dripper.Dripper
}

type CheckStatus struct {
	Check        *HttpCheck
	Message      string
	ResponseTime time.Duration
}

var wg sync.WaitGroup
var timeout = time.Duration(6 * time.Second)

func NewHttpEngine() *HttpEngine {
	return &HttpEngine{
		dripper: dripper.NewDripper(),
	}
}

func (e *HttpEngine) AddCheck() {

}

func printStatus(c chan *CheckStatus) {
	for {
		status := <-c
		fmt.Print(status.Check.Host)
		fmt.Println(status)

		c, err := raidman.Dial("tcp", "localhost:5555")
		if err != nil {
			panic(err)
		}

		var event = &raidman.Event{
			State:   status.Message,
			Host:    status.Check.Host,
			Service: "http",
			Metric:  status.ResponseTime.Seconds() * 1000,
		}

		err = c.Send(event)
		if err != nil {
			panic(err)
		}
	}
}

func collectDrops(faucet chan interface{}, codes chan *CheckStatus) {
	for {
		check := <-faucet
		go checkStatus(check.(*HttpCheck), codes)
	}
}

func checkStatus(check *HttpCheck, codes chan *CheckStatus) {
	if check.Timeout == 0 {
		check.Timeout = 10
	}
	timeout := time.Duration(check.Timeout) * time.Second

	transport := http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			con, err := net.DialTimeout(network, check.Address, timeout)
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
	resp, err := client.Head("http://" + check.Host + check.Path)
	if err == nil {
		end := time.Now()
		responseTime := end.Sub(start)
		httpStatus := &CheckStatus{check, strconv.Itoa(resp.StatusCode), responseTime}
		codes <- httpStatus
		defer resp.Body.Close()
	} else {
		end := time.Now()
		responseTime := end.Sub(start)
		httpStatus := &CheckStatus{check, err.Error(), responseTime}
		codes <- httpStatus
	}

	_ = err
	_ = resp
}
