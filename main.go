package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/etcd/client"
)

var (
	blfile = flag.String("blacklist", "", "blacklist file")
	etcd   = flag.Bool("etcd", false, "blacklist etcd")

	resOK = []byte("OK")
)

type blacklist map[string]struct{}

const (
	errArg = iota + 1
	errInternal
)

func handle(bl blacklist) func(rw http.ResponseWriter, r *http.Request) {
	return func(rw http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) <= 1 {
			http.Error(rw, "bad request", http.StatusBadRequest)
			return
		}

		host := r.URL.Path[1:]
		if _, ok := bl[host]; ok {
			http.Error(rw, "blacklist", http.StatusBadRequest)
			return
		}

		rw.Write(resOK)
	}
}

func main() {
	var (
		bl  blacklist
		err error
	)

	flag.Parse()

	if len(*blfile) > 0 {
		bl, err = makeBlacklistFromFile(*blfile)
		if err != nil {
			log.Println(err)
			os.Exit(errArg)
		}
	}

	if *etcd {
		bl, err = makeBlacklistFromEtcd()
		if err != nil {
			log.Println(err)
			os.Exit(errArg)
		}
	}

	if len(bl) == 0 {
		log.Println("blacklist is required")
		os.Exit(errArg)
	}

	http.HandleFunc("/", handle(bl))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Println(err)
		os.Exit(errInternal)
	}
}

func makeBlacklistFromFile(file string) (blacklist, error) {
	r, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return genBlacklist(string(data)), nil
}

func makeBlacklistFromEtcd() (blacklist, error) {
	cfg := client.Config{
		Endpoints: []string{"http://127.0.0.1:2379"},
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		return nil, err
	}
	kapi := client.NewKeysAPI(c)

	resp, err := kapi.Get(context.Background(), "/blacklist", nil)
	if err != nil {
		return nil, err
	}
	return genBlacklist(resp.Node.Value), nil
}

func genBlacklist(src string) blacklist {
	bl := blacklist{}
	list := strings.Split(src, "\n")
	for _, s := range list {
		bl[s] = struct{}{}
	}

	return bl
}
