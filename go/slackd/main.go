package main

import (
	"slack"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/ardielle/ardielle-go/rdl"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"encoding/json"
	"regexp"
	"os/exec"
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

//
// SlackImpl is the implementation of the CapsHandler interface
//
type SlackImpl struct {
	baseUrl  string
}

// Implementation
func (impl *SlackImpl) PostRequest(context *rdl.ResourceContext, request *slack.Request) (*slack.Request, error) {
	canonicalStr, err := json.Marshal(request)
	r := regexp.MustCompile(`^<(.*?)\|`)
	if request.Event != nil && request.Event.Text != "" {
		result := r.FindAllStringSubmatch(request.Event.Text, -1)
		if result != nil && 2 <= len(result[0]) {
			log.Printf("%s", result[0][1])
		}
	}
	if err != nil {
		errMsg := fmt.Sprintf("Failed to Marshal Json for converting request to canonical form, Error:", err)
		log.Printf(string(errMsg))
		return request, &rdl.ResourceError{Code: 200, Message: errMsg}
	}
	log.Printf("%s", string(canonicalStr))
	return request, nil
}

// Implementation
func (impl *SlackImpl) GetNgrokInterface(context *rdl.ResourceContext) (*slack.NgrokInterface, error) {
	cmdstr := "echo -n $(route | awk 'NR==3 {print $2}')"
	out, err := exec.Command("bash", "-c", cmdstr).Output()
	if err != nil {
		errMsg := fmt.Sprintf("Unable to execute command, Error: %v", err)
		log.Printf("%s", errMsg)
		return nil, &rdl.ResourceError{Code: 500, Message: errMsg}
	}
	log.Printf("Output: %s", string(out))
	if string(out) == "" {
		errMsg := fmt.Sprintf("No output from command, Error: %s", string(out))
		log.Printf("%s", errMsg)
		return nil, &rdl.ResourceError{Code: 500, Message: errMsg}
	}

	client := slack.NewClient("http://" + string(out) + ":4040", nil)
	ngrokif, err := client.GetNgrokInterface()
	if err != nil {
		errMsg := fmt.Sprintf("Unable to retrieve ngrok response details for GetNgrokInterface, Error: %v", err)
		log.Printf("%s", errMsg)
		return nil, &rdl.ResourceError{Code: 500, Message: errMsg}
	}
	return ngrokif, nil
}

// Implementation
func (impl *SlackImpl) PostWebhookRequest(context *rdl.ResourceContext, T string, B string, X string, request *slack.WebhookRequest) (slack.WebhookResponse, error) {
	ngrokif, err := impl.GetNgrokInterface(context)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to retrieve ngrok response details for PostWebhookRequest, Error: %v", err)
		log.Printf("%s", errMsg)
		return slack.WebhookResponse(request.Text), &rdl.ResourceError{Code: 500, Message: errMsg}
	}
	request.Text = ngrokif.Public_url + "/api/v1/services/" + T + "/" + B + "/" + X
	log.Printf("%s", request.Text)

	slackClient := slack.NewClient("https://hooks.slack.com", getHttpTransport())
	response, err := slackClient.PostWebhookRequest(T, B, X, request)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to retrieve slack response details for PostWebhookRequest, Error: %v", err)
		log.Printf("%s", errMsg)
	}
	return slack.WebhookResponse(response), nil
}

// Implementation
func (impl *SlackImpl) GetWebhookResponse(context *rdl.ResourceContext, T string, B string, X string) (slack.WebhookResponse, error) {
	return impl.PostWebhookRequest(context, T, B, X, new(slack.WebhookRequest))
}

func getHttpTransport() *http.Transport {
	config := &tls.Config{}
	config.InsecureSkipVerify = true
	tr := http.Transport{}
	tr.TLSClientConfig = config
	return &tr
}

//
// the following is to support TLS-based authentication, and self-authorization that just logs what if *could* enforce.
//

func (impl *SlackImpl) Authorize(action string, resource string, principal rdl.Principal) (bool, error) {
	fmt.Printf("[Authorize '%v' to %v on %v]\n", principal, action, resource)
	return true, nil
}

func (impl *SlackImpl) Authenticate(context *rdl.ResourceContext) bool {
	certs := context.Request.TLS.PeerCertificates
	for _, cert := range certs {
		fmt.Printf("[Authenticated '%s' from TLS client cert]\n", cert.Subject.CommonName)
		context.Principal = &TLSPrincipal{cert}
		return true
	}
	return false
}

type TLSPrincipal struct {
	Cert *x509.Certificate
}

func (p *TLSPrincipal) String() string {
	return p.GetYRN()
}

func (p *TLSPrincipal) GetDomain() string {
	cn := p.Cert.Subject.CommonName
	i := strings.LastIndex(cn, ".")
	return cn[0:i]
}

func (p *TLSPrincipal) GetName() string {
	cn := p.Cert.Subject.CommonName
	i := strings.LastIndex(cn, ".")
	return cn[i+1:]
}

func (p *TLSPrincipal) GetYRN() string {
	return p.Cert.Subject.CommonName
}

func (p TLSPrincipal) GetCredentials() string {
	return ""
}

func (p TLSPrincipal) GetHTTPHeaderName() string {
	return ""
}

func TLSConfiguration() (*tls.Config, error) {
	capem, err := ioutil.ReadFile("certs/ca.cert")
	if err != nil {
		return nil, err
	}
	config := &tls.Config{}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(capem) {
		return nil, fmt.Errorf("Failed to append certs to pool")
	}
	config.RootCAs = certPool

	keypem, err := ioutil.ReadFile("keys/slack.key")
	if err != nil {
		return nil, err
	}
	certpem, err := ioutil.ReadFile("certs/slack.cert")
	if err != nil {
		return nil, err
	}
	if certpem != nil && keypem != nil {
		mycert, err := tls.X509KeyPair(certpem, keypem)
		if err != nil {
			return nil, err
		}
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0] = mycert

		config.ClientCAs = certPool

		//config.ClientAuth = tls.RequireAndVerifyClientCert
		config.ClientAuth = tls.VerifyClientCertIfGiven
	}

	//Use only modern ciphers
	config.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}

	//Use only TLS v1.2
	config.MinVersion = tls.VersionTLS12

	//Don't allow session resumption
	config.SessionTicketsDisabled = true
	return config, nil

}
