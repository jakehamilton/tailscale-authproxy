package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	// "os"
	"strings"
	"tailscale.com/client/tailscale"
)

var (
	listenType        = flag.String("listen-type", "tcp", "whether to listen via tcp or unix socket")
	tcpAddr           = flag.String("tcp-addr", "127.0.0.1:8001", "the address to listen on with tcp")
	socketPath        = flag.String("socket-path", "", "the path to the unix socket to listen on")
	allowUnauthorized = flag.Bool("allow-unauthorized", false, "whether to allow unauthorized machines")
	restrictTailnet   = flag.String("restrict-tailnet", "", "limit access to a single tailnet")
)

func main() {
	flag.Parse()

	log.Printf("listenType=%s tcpAddr=%s socketPath=%s\n", *listenType, *tcpAddr, *socketPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		remoteHost := r.Header.Get("Remote-Addr")
		remotePort := r.Header.Get("Remote-Port")

		if remoteHost == "" || remotePort == "" {
			w.WriteHeader(http.StatusBadRequest)
			log.Println("set Remote-Addr to $remote_addr and Remote-Port to $remote_port in your nginx config")
			return
		}

		remoteAddrStr := net.JoinHostPort(remoteHost, remotePort)
		remoteAddr, err := netip.ParseAddrPort(remoteAddrStr)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("remote address and port are not valid: %v", err)
			return
		}

		info, err := tailscale.WhoIs(r.Context(), remoteAddr.String())
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			log.Printf("can't look up %s: %v", remoteAddr, err)
			return
		}

		// tailnet of connected node. When accessing shared nodes, this
		// will be empty because the tailnet of the sharee is not exposed.
		var tailnet string

		if !info.Node.Hostinfo.ShareeNode() {
			var ok bool
			_, tailnet, ok = strings.Cut(info.Node.Name, info.Node.ComputedName+".")
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				log.Printf("can't extract tailnet name from hostname %q", info.Node.Name)
				return
			}
			tailnet = strings.TrimSuffix(tailnet, ".beta.tailscale.net")
		}

		if *restrictTailnet != "" && *restrictTailnet != tailnet {
			w.WriteHeader(http.StatusForbidden)
			log.Printf("user is part of tailnet %s, wanted: %s", tailnet, url.QueryEscape(*restrictTailnet))
			return
		}

		log.Printf(
			"login=%v user=%v name=%v tailnet=%v node=%v authorized=%v tags=%v",
			strings.Split(info.UserProfile.LoginName, "@")[0],
			info.UserProfile.LoginName,
			info.UserProfile.DisplayName,
			tailnet,
			info.Node.DisplayName(true),
			info.Node.MachineAuthorized,
			strings.Join(info.Node.Tags, ","),
		)

		user := info.UserProfile.LoginName

		nodeGroup := "node-" + info.Node.DisplayName(true)
		groups := []string{nodeGroup}

		if tailnet != "" {
			groups = append(groups, "tailnet-"+tailnet)
		}

		if info.Node.MachineAuthorized {
			groups = append(groups, "authorized")
		} else {
			if !*allowUnauthorized {
				w.WriteHeader(http.StatusUnauthorized)
				log.Printf("unauthorized machine denied %q", info.Node.Name)
				return
			}

			groups = append(groups, "unauthorized")
		}

		for _, tag := range info.Node.Tags {
			if strings.HasPrefix(tag, "tag:dex-group-") {
				groups = append(groups, strings.TrimPrefix(tag, "tag:dex-group-"))
			}

			if strings.HasPrefix(tag, "tag:dex-user-") {
				user = strings.TrimPrefix(tag, "tag:dex-user-")
				user = strings.ReplaceAll(user, "--dot--", ".")
				user = strings.ReplaceAll(user, "--at--", "@")
			}
		}

		h := w.Header()

		h.Set("X-Remote-User", user)
		h.Set("X-Remote-Groups", strings.Join(groups, ","))

		w.WriteHeader(http.StatusNoContent)
	})

	var ln net.Listener
	var err error

	if *listenType == "tcp" {
		ln, err = net.Listen("tcp", *tcpAddr)
	} else {
		ln, err = net.Listen("unix", *socketPath)
	}

	if err != nil {
		log.Fatalf("could not start server on port: %v", err)
	}

	http.Serve(ln, mux)

	for {
		select {}
	}
}
