package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/suzuki-shunsuke/go-graylog"
	"github.com/suzuki-shunsuke/go-graylog/client"
	"github.com/suzuki-shunsuke/go-set"
)

var (
	gl_url      = os.Getenv("GRAYLOG_URL")
	gl_user     = os.Getenv("GRAYLOG_USER")
	gl_password = os.Getenv("GRAYLOG_PASSWORD")
)

func main() {
	Provision(context.Background(), "testproj", "addq2eq13123123")
}

func provisionUser(ctx context.Context, client *client.Client, username string, password string) (string, error) {
	// check user existance
	users, _, err := client.GetUsersContext(ctx)
	if err != nil {
		return "", err
	}

	for _, user := range users {
		if user.Username == username {
			log.Printf("User '%s' already provisioned. ID=%s", username, user.ID)
			return user.ID, nil
		}
	}

	// log.Print(users, ei, err)

	// create the user

	user := graylog.User{
		Username: username,
		FullName: username,
		Password: password,
		Email:    username + "@wd-logging-operator",
		Roles:    set.NewStrSet("Reader", "Dashboard Creator"),
	}

	_, err = client.CreateUserContext(ctx, &user)
	if err != nil {
		log.Fatalf("Error creating user '%s': %d", username, err)
		return "", err
	}

	log.Printf("User '%s' created", username)

	return user.ID, nil
}

func findIndexSetByTitle(ctx context.Context, client *client.Client, title string) (*graylog.IndexSet, error) {
	sets, _, _, _, err := client.GetIndexSetsContext(ctx, 0, 0, false)
	if err != nil {
		return nil, err
	}

	for _, set := range sets {
		if set.Title == title {
			return &set, nil
		}
	}

	return nil, nil
}

func provisionIndexSet(ctx context.Context, client *client.Client, name string, templateName string) (string, error) {

	// check existance
	indexset, err := findIndexSetByTitle(ctx, client, name)
	if err != nil {
		return "", err
	}

	if indexset != nil {
		log.Printf("IndexSet '%s' already exists. ID=%s", name, indexset.ID)
		return indexset.ID, nil
	}

	// get the template
	template, err := findIndexSetByTitle(ctx, client, templateName)
	if err != nil {
		return "", err
	}

	if template == nil {
		return "", errors.New("Template IndexSet '" + templateName + "' not found")
	}

	// update the fields
	indexset = template
	indexset.ID = ""
	indexset.Title = name
	indexset.IndexPrefix = name + "-"
	indexset.Description = name + "@wd-logging-operator"

	// create it
	_, err = client.CreateIndexSetContext(ctx, indexset)
	if err != nil {
		log.Fatalf("Error creating IndexSet '%s': %d", name, err)
		return "", err
	}

	log.Printf("IndexSet '%s' created. ID=%s", name, indexset.ID)
	return indexset.ID, nil
}

func provisionStream(ctx context.Context, client *client.Client, name string, ruleFieldName string, ruleFieldValue string, userID string, indexSetID string) (string, error) {

	// check existance
	streams, _, _, err := client.GetStreams()
	if err != nil {
		return "", err
	}

	for _, stream := range streams {
		if stream.Title == name {
			log.Printf("Stream '%s' already exists. ID=%s", name, stream.ID)
			return stream.ID, nil
		}
	}

	// create the stream
	stream := graylog.Stream{
		Title:                          name,
		Description:                    name + "@wd-logging-operator",
		IndexSetID:                     indexSetID,
		RemoveMatchesFromDefaultStream: true,
		Rules: []graylog.StreamRule{
			{
				Field: ruleFieldName,
				Value: ruleFieldValue,
				Type:  1,
			},
		},
	}

	_, err = client.CreateStreamContext(ctx, &stream)
	if err != nil {
		log.Fatalf("Error creating Stream '%s': %s", name, err)
		return "", err
	}

	log.Printf("Stream '%s' created. ID=%s", name, stream.ID)

	// start the stream
	_, err = client.ResumeStreamContext(ctx, stream.ID)
	if err != nil {
		log.Fatalf("Error starting Stream '%s': %s", name, err)
		return "", err
	}

	log.Printf("Stream '%s' started", name)

	// share the stream
	// share is currently not implemented (see https://github.com/suzuki-shunsuke/go-graylog/issues/332), so we call the API by ourself

	/* By inspecting the Rest Calls from the Graylog UI, we see the following POST call executed:

	First there is a POST call to $GRAYLOG/api/authz/shares/entities/grn::::stream:60a242439e82ee1814ce2cd5/prepare, but it seems to be not
	mandatory, as we can create shares successfully with curl without `/prepare`.

	This is the call on "Save":
	curl "$GRAYLOG/api/authz/shares/entities/grn::::stream:60a242439e82ee1814ce2cd5" \
		  --data-raw '{"selected_grantee_capabilities":{"grn::::user:60a226a99e82ee1814ce0e92":"view"}}'

	60a242439e82ee1814ce2cd5 is the ID of the Stream
	60a226a99e82ee1814ce0e92 is the ID of the User

	*/

	grn := "grn::::stream:" + stream.ID

	request := struct {
		Selected_grantee_capabilities map[string]interface{} `json:"selected_grantee_capabilities,omitempty"`
	}{map[string]interface{}{
		"grn::::user:" + userID: "view",
	}}

	_, err = CallAPI(client, ctx, http.MethodPost, gl_url+"/authz/shares/entities/"+grn, request, nil)
	if err != nil {
		log.Fatalf("Error sharing Stream '%s': %s", name, err)
		return "", err
	}

	log.Printf("Stream '%s' shared", name)

	return stream.ID, nil
}

func Provision(ctx context.Context, name string, password string) error {
	cl, err := client.NewClientV3(gl_url, gl_user, gl_password)
	if err != nil {
		return err
	}

	userId, err := provisionUser(ctx, cl, name, password)
	if err != nil {
		return err
	}

	indexSetId, err := provisionIndexSet(ctx, cl, name, "wd-logging-operator-template")
	if err != nil {
		return err
	}

	streamId, err := provisionStream(ctx, cl, name, "kubernetes_namespace_name", "testproj", userId, indexSetId)
	if err != nil {
		return err
	}

	_ = streamId
	return nil
}

func CallAPI(self *client.Client,
	ctx context.Context, method, endpoint string, input, output interface{},
) (*client.ErrorInfo, error) {
	// prepare request
	var (
		req *http.Request
		err error
	)
	if input != nil {
		reqBody := &bytes.Buffer{}
		if err := json.NewEncoder(reqBody).Encode(input); err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		req, err = http.NewRequest(method, endpoint, reqBody)
	} else {
		req, err = http.NewRequest(method, endpoint, nil)
	}
	if err != nil {
		return nil, fmt.Errorf(
			"failed to call http.NewRequest: %s %s: %w", method, endpoint, err)
	}
	ei := &client.ErrorInfo{Request: req}
	req.SetBasicAuth(self.Name(), self.Password())
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	// https://github.com/suzuki-shunsuke/go-graylog/issues/42
	req.Header.Set("X-Requested-By", "client-go")
	hc := http.DefaultClient

	// request
	resp, err := hc.Do(req)
	if err != nil {
		return ei, fmt.Errorf(
			"failed to call Graylog API: %s %s: %w", method, endpoint, err)
	}
	defer resp.Body.Close()
	ei.Response = resp

	if resp.StatusCode >= 400 {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return ei, fmt.Errorf(
				"graylog API error: failed to read the response body: %s %s %d",
				method, endpoint, resp.StatusCode)
		}
		if err := json.Unmarshal(b, ei); err != nil {
			return ei, fmt.Errorf(
				"failed to parse response body as ErrorInfo: %s %s %d %s: %w",
				method, endpoint, resp.StatusCode, string(b), err)
		}
		return ei, fmt.Errorf(
			"graylog API error: %s %s %d: "+string(b),
			method, endpoint, resp.StatusCode)
	}
	if output != nil {
		if err := json.NewDecoder(ei.Response.Body).Decode(output); err != nil {
			return ei, fmt.Errorf(
				"failed to decode graylog API response body: %s %s: %w",
				method, endpoint, err)
		}
	}
	return ei, nil
}
