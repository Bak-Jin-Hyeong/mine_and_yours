package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
)

type Addr struct {
	Network string
	String  string
}

type NIC struct {
	Index         int
	MTU           int
	Name          string
	HardwareAddr  string
	Flags         string
	Addr          []Addr
	MulticastAddr []Addr
}

func toNIC(i net.Interface) (NIC, error) {
	nic := NIC{
		Index:        i.Index,
		MTU:          i.MTU,
		Name:         i.Name,
		HardwareAddr: i.HardwareAddr.String(),
		Flags:        i.Flags.String(),
	}

	var err error

	addrs, err0 := i.Addrs()
	if err0 != nil {
		log.Println("NIC", nic, "Addrs Error:", err0)
	}

	for _, a := range addrs {
		nic.Addr = append(nic.Addr, Addr{a.Network(), a.String()})
	}

	multicastAddrs, err1 := i.MulticastAddrs()
	if err1 != nil {
		log.Println("NIC", nic, "MulticastAddrs Error:", err1)
	}

	for _, a := range multicastAddrs {
		nic.MulticastAddr = append(nic.MulticastAddr, Addr{a.Network(), a.String()})
	}

	err = err0
	if err == nil {
		err = err1
	}
	return nic, err
}

func getNICs() ([]NIC, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println("net.Interfaces Error:", err)
		return nil, err
	}

	var ret []NIC
	for _, i := range interfaces {
		nic, e := toNIC(i)
		ret = append(ret, nic)
		if err == nil && e != nil {
			err = e
		}
	}
	return ret, nil
}

type Short struct {
	Mine string
	Your string
}

type Verbose struct {
	Short
	MyNIC []NIC
}

type contextKey struct {
	key string
}

var ConnContextKey = &contextKey{"http-conn"}

func saveConnInContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, ConnContextKey, c)
}
func getConn(r *http.Request) net.Conn {
	return r.Context().Value(ConnContextKey).(net.Conn)
}

type handler struct {
	srv *http.Server
	nic []NIC
}

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn := getConn(req)
	localAddr := conn.LocalAddr().String()
	code := http.StatusOK

	defer func() {
		xf := req.Header["X-Forwarded-For"]
		xi := req.Header["X-Real-IP"]
		ua := req.Header["User-Agent"]

		log.Println(req.Proto, req.Method, req.Host, req.URL.String(), code, req.RemoteAddr, xf, xi, ua, localAddr)
	}()

	switch req.Method {
	case http.MethodGet:
	case http.MethodHead:
	case http.MethodOptions:
		w.Header().Add("Allow", http.MethodGet)
		w.Header().Add("Allow", http.MethodHead)
		w.Header().Add("Allow", http.MethodOptions)
		w.WriteHeader(http.StatusOK)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	writeResponse := func(code int, b []byte) {
		if req.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(b)))
			w.WriteHeader(code)
			return
		}
		w.WriteHeader(code)
		_, _ = w.Write(b)
	}

	short := Short{}
	short.Mine = localAddr
	short.Your = req.RemoteAddr

	if req.URL.Path == "/" {
		shortB, err := json.MarshalIndent(short, "", " ")
		if err != nil {
			code = http.StatusInternalServerError
			w.WriteHeader(code)
			return
		}
		writeResponse(code, shortB)
		return
	}

	verbose := Verbose{Short: short}
	verbose.MyNIC = h.nic
	verboseB, err := json.MarshalIndent(verbose, "", " ")
	if err != nil {
		code = http.StatusInternalServerError
		w.WriteHeader(code)
		return
	}

	writeResponse(code, verboseB)
}

func main() {
	log.Println("init")
	defer log.Println("term")

	nic, err := getNICs()
	if err != nil {
		log.Println("getNICs:", err)
	}

	b, err := json.MarshalIndent(nic, "", " ")
	if err != nil {
		log.Println("json.MarshalIndent:", err)
	}

	_ = os.Stderr.Sync()
	_, _ = os.Stderr.Write([]byte("\n"))
	_, _ = os.Stderr.Write(b)
	_, _ = os.Stderr.Write([]byte("\n"))
	_ = os.Stderr.Sync()

	var listen string
	flag.StringVar(&listen, "listen", "localhost:80", "listen address")
	flag.Parse()
	if listen == "" {
		flag.Usage()
		os.Exit(1)
		return
	}

	log.Println("try listen:", listen)
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		log.Println("listen:", err)
		os.Exit(1)
		return
	}
	log.Println("listen:", listen)

	h := &handler{nic: nic}
	srv := &http.Server{
		ConnContext: saveConnInContext,
	}
	h.srv = srv
	srv.Handler = h

	err = srv.Serve(ln)
	log.Println(err)
}
