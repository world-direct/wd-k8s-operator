package graylog

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type GraylogClient struct {
	Url      string
	User     string
	Password string
	Log      logr.Logger
}

type GraylogProvisioningData struct {

	// Name is used as the Name / Title for all provisioned objects
	Name string

	// User data
	User struct {
		InitialPassword string
		Roles           []string

		id string
	}

	// IndexSet data
	IndexSet struct {
		TemplateName string

		id string
	}

	// Stream data
	Stream struct {

		// The FieldName to match 'Name'
		RuleFieldName string

		id string
	}
}

func (client GraylogClient) Test(ctx context.Context) error {
	return client.callAPIExpect(ctx, "GET", "/api/cluster", nil, nil, 200)
}

func (client GraylogClient) callAPIExpect(ctx context.Context, method, endpoint string, input, output interface{}, expectedStatusCode int) error {
	sc, err := client.callAPI(ctx, method, endpoint, input, output)
	if err != nil {
		// no Wrap here, because we have the context information in callAPI
		return err
	}

	if sc != expectedStatusCode {
		return errors.Errorf("%s %s status %d, %d expected", method, endpoint, sc, expectedStatusCode)
	}

	return nil
}

func (client GraylogClient) callAPI(ctx context.Context, method, endpoint string, input, output interface{}) (int, error) {

	// prepare request
	var (
		req *http.Request
		err error
	)

	log := client.Log.WithValues("method", method, "endpoint", endpoint)

	// build request
	reqBody := &bytes.Buffer{}
	if input != nil {
		if err := json.NewEncoder(reqBody).Encode(input); err != nil {
			return 0, errors.Wrap(err, "failed to encode request body")
		}
	}

	// make URL
	url := client.Url + endpoint

	req, err = http.NewRequest(method, url, reqBody)
	if err != nil {
		return 0, errors.Wrap(err, "failed to call http.NewRequest")
	}

	req = req.WithContext(ctx)

	req.SetBasicAuth(client.User, client.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-By", "wd-k8s-operator")

	// request
	hc := &http.Client{}
	resp, err := hc.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "failed to execute http request")
	}

	defer resp.Body.Close()

	// read response to buffer so that we can log the body an decode the output
	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, errors.Wrap(err, "failed to read response body")
	}

	log.V(2).Info("GrayLogAPICall", "StatusCode", resp.StatusCode, "Body", string(responseBody))

	if output != nil {
		if err := json.NewDecoder(bytes.NewReader(responseBody)).Decode(output); err != nil {
			return resp.StatusCode, errors.Wrap(err, "failed to decode graylog API response body")
		}
	}

	return resp.StatusCode, nil
}

// returns a Client instance for API calls.
// Reads GRAYLOG_URL, GRAYLOG_USER, GRAYLOG_PASSWORD environment variables
func CreateClient(log logr.Logger) (GraylogClient, error) {

	client := GraylogClient{
		Url:      os.Getenv("GRAYLOG_URL"),
		User:     os.Getenv("GRAYLOG_USER"),
		Password: os.Getenv("GRAYLOG_PASSWORD"),
		Log:      log,
	}

	if client.Url == "" {
		return client, errors.New("Missing GRAYLOG_URL enviornment variable")
	}

	if client.User == "" {
		return client, errors.New("Missing GRAYLOG_USER enviornment variable")
	}

	if client.Password == "" {
		return client, errors.New("Missing GRAYLOG_PASSWORD enviornment variable")
	}

	return client, nil

}

const OPERATOR_INFO = "wd-k8s-operator"
const STREAM_TEMPLATE_NAME = "wd-k8s-operator-template"
