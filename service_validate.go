package cas

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/golang/glog"
)

func NewServiceTicketValidator(client *http.Client, casUrl *url.URL) *ServiceTicketValidator {
	return &ServiceTicketValidator{
		client: client,
		casUrl: casUrl,
	}
}

// ServiceTicketValidator is responsible for the validation of a service ticket
type ServiceTicketValidator struct {
	client *http.Client
	casUrl *url.URL
}

// ValidateTicket validates the service ticket for the given server. The method will try to use the service validate
// endpoint of the cas >= 2 protocol, if the service validate endpoint not available, the function will use the cas 1
// validate endpoint.
func (validator *ServiceTicketValidator) ValidateTicket(serviceUrl *url.URL, ticket string) (*AuthenticationResponse, error) {
	if glog.V(2) {
		glog.Infof("Validating ticket %v for service %v", ticket, serviceUrl)
	}

	u, err := validator.ServiceValidateUrl(serviceUrl, ticket)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("User-Agent", "Golang CAS client gopkg.in/cas")

	if glog.V(2) {
		glog.Infof("Attempting ticket validation with %v", r.URL)
	}

	resp, err := validator.client.Do(r)
	if err != nil {
		return nil, err
	}

	if glog.V(2) {
		glog.Infof("Request %v %v returned %v",
			r.Method, r.URL,
			resp.Status)
	}

	if resp.StatusCode == http.StatusNotFound {
		return validator.validateTicketCas1(serviceUrl, ticket)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cas: validate ticket: %v", string(body))
	}

	if glog.V(2) {
		glog.Infof("Received authentication response\n%v", string(body))
	}

	body = []byte(strings.Replace(string(body), "[Etc/UTC]", "", -1))
	success, err := ParseServiceResponse(body)
	if err != nil {
		return nil, err
	}

	if glog.V(2) {
		glog.Infof("Parsed ServiceResponse: %#v", success)
	}

	return success, nil
}

// ServiceValidateUrl creates the service validation url for the cas >= 2 protocol.
// TODO the function is only exposed, because of the clients ServiceValidateUrl function
func (validator *ServiceTicketValidator) ServiceValidateUrl(serviceUrl *url.URL, ticket string) (string, error) {
	u, err := validator.casUrl.Parse(path.Join(validator.casUrl.Path, "serviceValidate"))
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(serviceUrl))
	q.Add("ticket", ticket)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (validator *ServiceTicketValidator) validateTicketCas1(serviceUrl *url.URL, ticket string) (*AuthenticationResponse, error) {
	u, err := validator.ValidateUrl(serviceUrl, ticket)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("User-Agent", "Golang CAS client gopkg.in/cas")

	if glog.V(2) {
		glog.Infof("Attempting ticket validation with %v", r.URL)
	}

	resp, err := validator.client.Do(r)
	if err != nil {
		return nil, err
	}

	if glog.V(2) {
		glog.Infof("Request %v %v returned %v",
			r.Method, r.URL,
			resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, err
	}

	body := string(data)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cas: validate ticket: %v", body)
	}

	if glog.V(2) {
		glog.Infof("Received authentication response\n%v", body)
	}

	if body == "no\n\n" {
		return nil, nil // not logged in
	}

	success := &AuthenticationResponse{
		User: body[4 : len(body)-1],
	}

	if glog.V(2) {
		glog.Infof("Parsed ServiceResponse: %#v", success)
	}

	return success, nil
}

// ValidateUrl creates the validation url for the cas >= 1 protocol.
// TODO the function is only exposed, because of the clients ValidateUrl function
func (validator *ServiceTicketValidator) ValidateUrl(serviceUrl *url.URL, ticket string) (string, error) {
	u, err := validator.casUrl.Parse(path.Join(validator.casUrl.Path, "validate"))
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Add("service", sanitisedURLString(serviceUrl))
	q.Add("ticket", ticket)
	u.RawQuery = q.Encode()

	return u.String(), nil
}
