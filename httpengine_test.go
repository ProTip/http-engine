package httpengine

import "testing"
import "github.com/protip/dripper"
import "bytes"
import "io/ioutil"

func TestHttpEngine(t *testing.T) {
	var codes = make(chan *CheckStatus, 50)
	d := dripper.NewDripper()
	urlsBytes, err := ioutil.ReadFile("urls.txt")
	_ = err
	urlsBytes = urlsBytes[:len(urlsBytes)-1]
	urls := bytes.Split(urlsBytes, []byte("\n"))
	for _, url := range urls {
		parts := bytes.Split(url, []byte(" "))
		d.AddDrop(string(url), &HttpCheck{
			Address: string(parts[0]),
			Host:    string(parts[1]),
			Path:    string(parts[2]),
		})
	}
	d.Drip()
	wg.Add(1)
	go collectDrops(d.Faucet, codes)
	go printStatus(codes)
	wg.Wait()
}
