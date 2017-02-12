// Copyright © 2017 Pennock Tech, LLC.
// All rights reserved, except as granted under license.
// Licensed per file LICENSE.txt

// We are canonically imported from go.pennock.tech/fingerd but because we are
// not a library, we do not apply this as an import constraint on the package
// declarations.  You can fork and build elsewhere more easily this way, while
// still getting dependencies without a dependency manager in play.
//
// This comment is just to let you know that the canonical import path is
// go.pennock.tech/fingerd and not now, nor ever, using DNS pointing to a
// code-hosting site not under our direct control.  We keep our options open,
// for moving where we keep the code publicly available.

package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
)

var opts struct {
	defaultPort  string
	tlsOnConnect bool
	showVersion  bool
}

type programStatus struct {
	probing    *sync.WaitGroup
	errorCount uint32 // only access via sync/atomic while go-routines running
	output     chan<- string
}

func init() {
	flag.StringVar(&opts.defaultPort, "port", "smtp(25)", "port to connect to")
	flag.BoolVar(&opts.tlsOnConnect, "tls-on-connect", false, "start TLS immediately upon connection")
	flag.BoolVar(&opts.showVersion, "version", false, "show version and exit")
}

func main() {
	flag.Parse()
	if opts.showVersion {
		version()
		return
	}

	hostlist := flag.Args()
	if len(hostlist) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	messages := make(chan string, 10)
	shuttingDown := &sync.WaitGroup{}
	go emitOutputMessages(messages, shuttingDown)

	status := &programStatus{
		probing: &sync.WaitGroup{},
		output:  messages,
	}

	for _, hostSpec := range hostlist {
		status.probing.Add(1)
		go probeHost(hostSpec, status)
	}

	status.probing.Wait()
	shuttingDown.Add(1)
	close(messages)
	shuttingDown.Wait()

	if status.errorCount != 0 {
		fmt.Fprintf(os.Stderr, "%s: encountered %d errors\n", os.Args[0], status.errorCount)
		os.Exit(1)
	}

	os.Exit(0)
}
