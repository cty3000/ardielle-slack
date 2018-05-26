package main

import (
	"slack"
	"crypto/tls"
	"fmt"
	"github.com/ardielle/ardielle-go/rdl"
	"log"
	"net/http"
	"encoding/json"
	"regexp"
	"os/exec"
)

//
// SlackImpl is the implementation of the CapsHandler interface
//
type SlackImpl struct {
	baseUrl  string
}

// Implementation
func (impl *SlackImpl) PostSlackEvent(context *rdl.ResourceContext, request *slack.SlackEvent) (*slack.SlackEvent, error) {
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
func (impl *SlackImpl) PostSlackWebhookRequest(context *rdl.ResourceContext, T string, B string, X string, request *slack.SlackWebhookRequest) (slack.SlackWebhookResponse, error) {
	ngrokif, err := impl.GetNgrokInterface(context)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to retrieve ngrok response details for PostSlackWebhookRequest, Error: %v", err)
		log.Printf("%s", errMsg)
		return slack.SlackWebhookResponse(request.Text), &rdl.ResourceError{Code: 500, Message: errMsg}
	}
	request.Text = ngrokif.Public_url + "/api/v1/services/" + T + "/" + B + "/" + X
	log.Printf("%s", request.Text)

	slackClient := slack.NewClient("https://hooks.slack.com", getHttpTransport())
	response, err := slackClient.PostSlackWebhookRequest(T, B, X, request)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to retrieve slack response details for PostSlackWebhookRequest, Error: %v", err)
		log.Printf("%s", errMsg)
	}
	return slack.SlackWebhookResponse(response), nil
}

// Implementation
func (impl *SlackImpl) GetSlackWebhookURL(context *rdl.ResourceContext, T string, B string, X string) (slack.SlackWebhookURL, error) {
	response, err := impl.PostSlackWebhookRequest(context, T, B, X, new(slack.SlackWebhookRequest))
	return slack.SlackWebhookURL(response), err
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
