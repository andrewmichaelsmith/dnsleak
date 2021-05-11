package main

import (
	"github.com/miekg/dns"
	"log"
	"strconv"
)

func parseQuery(m *dns.Msg) {

	for _, q := range m.Question {
		log.Printf("Query for %s\n", q.Name)
	}

}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	log.Printf("Query from %s\n", w.RemoteAddr())
	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m)
	}

	w.WriteMsg(m)
}

func main() {
	// attach request handler func
	dns.HandleFunc(".", handleDnsRequest)

	// start server
	port := 5354
	server := &dns.Server{Addr: ":" + strconv.Itoa(port), Net: "udp"}
	log.Printf("Starting at %d\n", port)
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}
}
