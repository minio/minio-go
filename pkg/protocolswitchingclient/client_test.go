package protocolswitchingclient

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/quic-go/quic-go/http3"
)

// create a self signed cert for unit test
func generateSelfSignedCert() (tlsCert string, tlsKey string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	certTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test Org"}},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}

	certOut := &bytes.Buffer{}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile, err := os.CreateTemp("", "cert-*.pem")
	if err != nil {
		log.Fatal(err)
	}

	certFile.Write(certOut.Bytes())

	// Don't forget to close it!
	if err := certFile.Close(); err != nil {
		log.Fatal(err)
	}

	keyOut := &bytes.Buffer{}
	pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyFile, err := os.CreateTemp("", "key-*.pem")
	if err != nil {
		log.Fatal(err)
	}

	keyFile.Write(keyOut.Bytes())

	// Don't forget to close it!
	if err := keyFile.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("cert", certFile.Name(), "key", keyFile.Name())
	return certFile.Name(), keyFile.Name(), nil
}

func TestTLSServer(t *testing.T) {
	createErrMsg := "unable to create new request %s"
	tlsCert, tlsKey, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate self-signed certificate: %v", err)
	}
	defer os.Remove(tlsCert)
	defer os.Remove(tlsKey)

	//port 4433 has a purposefully slower http3 response
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Proto == "HTTP/3" {
			time.Sleep(50 * time.Millisecond)
		}
		fmt.Fprintf(w, "Hello from %s!\n", r.Proto) // r.Proto: HTTP/1.1, HTTP/2, or HTTP/3
	}

	StartServer(4433, tlsCert, tlsKey, handler, true, t)

	//port 4432 has a purposefully delayed http1 connection
	handlerFasterhttp3 := func(w http.ResponseWriter, r *http.Request) {
		if r.Proto == "HTTP/1.1" {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Fprintf(w, "Hello from %s!\n", r.Proto) // r.Proto: HTTP/1.1, HTTP/2, or HTTP/3
	}
	//slower http response
	StartServer(4432, tlsCert, tlsKey, handlerFasterhttp3, true, t)

	//port 4431 has no http3 response, to simulate a firewall change
	StartServer(4431, tlsCert, tlsKey, handlerFasterhttp3, false, t)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	time.Sleep(time.Second)

	CheckRedirect := func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	dynamClient := NewDynamicClient(tr, nil, CheckRedirect, StateHttp3, 30*time.Second)

	req, err := http.NewRequest("GET", "https://minio-a:4433/", nil)
	if err != nil {
		t.Fatalf(createErrMsg, err)
	}
	http3response := "Hello from HTTP/3.0!\n"
	httpresponse := "Hello from HTTP/1.1!\n"

	//Test both http and http3 to see which one wins
	Call(req, dynamClient, http3response, t)

	//Test 'fast pass', starting with http3
	Call(req, dynamClient, http3response, t)

	dynamClient.Reset()
	//Test faster http3 connection
	req, err = http.NewRequest("GET", "https://minio-a:4432/", nil)
	if err != nil {
		t.Fatalf(createErrMsg, err)
	}
	Call(req, dynamClient, http3response, t)

	//Test dropped http3 connection
	req, err = http.NewRequest("GET", "https://minio-a:4431/", nil)
	if err != nil {
		t.Fatalf(createErrMsg, err)
	}

	//This one will try http3, fail, and try http
	Call(req, dynamClient, httpresponse, t)

	//This will try both, and set to http 'fast' if http3 doesn't work again
	Call(req, dynamClient, httpresponse, t)

	//This will only use http 'fast' path
	Call(req, dynamClient, httpresponse, t)
	dynamClient.Close()
}

// allows us to start web servers supporting different ports and http3/1 variations
func StartServer(port int, tlsCert, tlsKey string, handler func(w http.ResponseWriter, r *http.Request), enableHTTP3 bool, t *testing.T) {
	// Create a multiplexer to handle HTTP routes
	mux := http.NewServeMux()
	// Simple handler: returns a message including the HTTP protocol used
	mux.HandleFunc("/", handler)

	// ----- HTTP/1.1 & HTTP/2 over TCP -----
	// Go's standard library automatically supports HTTP/1.1 and HTTP/2 over TLS
	tcpSrv := &http.Server{
		Addr:    fmt.Sprintf("minio-a:%d", port), // TCP port for HTTPS
		Handler: mux,                             // Use the multiplexer defined above
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS13, // HTTP/3 requires TLS 1.3; use same for parity
		},
	}

	// Run TCP server in a goroutine to allow HTTP/3 server to run concurrently
	go func() {
		log.Println("Serving HTTP/1.1 and HTTP/2 on https://localhost:", port)
		if err := tcpSrv.ListenAndServeTLS(tlsCert, tlsKey); err != nil {
			log.Fatal(err)
		}
	}()

	//skip setting up http3 connection if we didn't want it
	if !enableHTTP3 {
		return
	}

	// ----- HTTP/3 over QUIC/UDP -----
	// Resolve UDP address for QUIC (HTTP/3) server
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port)) // HTTP/3 uses UDP
	if err != nil {
		t.Fatal(err)
	}
	// Listen on UDP port
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	// Create HTTP/3 server using the same mux
	h3Srv := http3.Server{
		Addr:      fmt.Sprintf(":%d", port),                                                // Port for QUIC
		Handler:   mux,                                                                     // Same handler for HTTP/1.1, HTTP/2, HTTP/3
		TLSConfig: &tls.Config{Certificates: loadCert(string(tlsCert), string(tlsKey), t)}, // TLS for QUIC
	}

	// Start HTTP/3 server
	log.Println("Serving HTTP/3 on https://localhost:", port)
	go func() {
		fmt.Println("starting http3 server")
		if err := h3Srv.Serve(udpConn); err != nil {
			log.Fatal(err)
		}
	}()
}

func Call(req *http.Request, dynamClient *DynamicClient, expected string, t *testing.T) {
	resp, err := dynamClient.Do(req)

	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Check the response body
	body := new(bytes.Buffer)
	body.ReadFrom(resp.Body)

	if body.String() != expected {
		t.Errorf("Expected body %q; got %q", expected, body.String())
	}
}

func loadCert(certFile, keyFile string, t *testing.T) []tls.Certificate {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	return []tls.Certificate{cert}
}
