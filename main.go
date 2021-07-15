package main

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/miekg/dns"
	"github.com/olivere/elastic"
)

// Tweet is a structure used for serializing/deserializing data in Elasticsearch.
type Record struct {
	Date       time.Time `json:"date"`
	Query      string    `json:"query"`
	SourceIP   string    `json:"source-ip"`
	SourcePort int       `json:"source-port"`
}

type Config struct {
	Elastic struct {
		Port        string `envconfig:"ES_PORT" default:"9200" required:"true"`
		Host        string `envconfig:"ES_HOST" default:"localhost" required:"true"`
		Username    string `envconfig:"ES_USERNAME"`
		Password    string `envconfig:"ES_PASSWORD"`
		VerifyCerts bool   `envconfig:"ES_VERIFY_CERTS" default:"true"`
	}
}

var ctx context.Context
var client *elastic.Client

func recordQuery(m *dns.Msg, sourceIP string, sourcePort int) {

	for _, q := range m.Question {
		log.Printf("Query from %s for %s %d %d\n", sourceIP, q.Name, q.Qclass, q.Qtype)
		record := Record{Date: time.Now(), Query: q.Name, SourceIP: sourceIP, SourcePort: sourcePort}
		_, err := client.Index().
			Index("dnsleak").
			Type("event").
			BodyJson(record).
			Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}

	}

}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		remoteAddr := strings.Split(w.RemoteAddr().String(), ":")

		port, err := strconv.Atoi(remoteAddr[1])
		if err != nil {
			log.Fatalf("Failed to connect to es: %s\n ", err.Error())
		}
		recordQuery(m, remoteAddr[0], port)
	}

	w.WriteMsg(m)
}

func main() {

	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		panic(err)
	}

	url := url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(cfg.Elastic.Host, cfg.Elastic.Port),
	}

	ctx = context.Background()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !cfg.Elastic.VerifyCerts},
	}
	httpClient := &http.Client{Transport: tr}

	client, err = elastic.NewClient(
		elastic.SetHttpClient(httpClient),
		elastic.SetURL(url.String()),
		elastic.SetSniff(false),
		elastic.SetBasicAuth(cfg.Elastic.Username, cfg.Elastic.Password),
	)

	if err != nil {
		log.Fatalf("Failed to connect to es: %s\n ", err.Error())
	}
	//client.index
	// attach request handler func
	dns.HandleFunc(".", handleDnsRequest)

	// start server
	port := 5354
	server := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}
	log.Printf("Starting at %d\n", port)
	err = server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}
}
