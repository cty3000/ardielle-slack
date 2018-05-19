package main

import (
	"slack"
	"crypto/tls"
	"github.com/ardielle/ardielle-go/rdl"
	"log"
	"net/http"
	"os"
	"strings"
)

func now() rdl.Timestamp {
	return rdl.TimestampNow()
}

func defaultEndPoint() string {
	h := os.Getenv("HOST")
	if h != "" {
		p := os.Getenv("PORT")
		if p != "" {
			return h + ":" + p
		}
		return h + ":4080"
	}

	p := os.Getenv("PORT")
	if p != "" {
		return "0.0.0.0:" + p
	}

	endpoint := "0.0.0.0:4080"
	return endpoint
}

func defaultURL() string {
	url := "http://" + defaultEndPoint() + "/api/v1"
	return url
}

func main() {
	endpoint := defaultEndPoint()
	url := defaultURL()

	impl := new(SlackImpl)
	impl.baseUrl = url

	handler := slack.Init(impl, url, impl)

	if strings.HasPrefix(url, "https") {
		config, err := TLSConfiguration()
		if err != nil {
			log.Fatal("Cannot set up TLS: " + err.Error())
		}
		listener, err := tls.Listen("tcp", endpoint, config)
		if err != nil {
			panic(err)
		}
		log.Fatal(http.Serve(listener, handler))
	} else {
		log.Fatal(http.ListenAndServe(endpoint, handler))
	}
}
