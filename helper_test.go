package httpbin_test

import (
	"io/ioutil"
	"log"
	"math/rand"
	"testing"

	"github.com/bwangelme/go-httpbin"
)

func TestResource(t *testing.T) {
	originData := make([]byte, 8984)
	_, err := rand.Read(originData)
	if err != nil {
		log.Fatalln(err)
	}
	fd, err := ioutil.TempFile("/tmp", "*.bin")

	if err != nil {
		log.Fatalln(err)
	}
	fd.Write(originData)
	fd.Close()

	data, err := httpbin.Resource(fd.Name())
	if err != nil {
		log.Fatalln(err)
	}
	if len(data) != len(originData) {
		log.Fatalf("%d != %d", len(data), len(originData))
	}
}
