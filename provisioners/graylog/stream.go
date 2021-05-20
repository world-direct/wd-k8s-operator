package graylog

import (
	"context"

	"github.com/go-logr/logr"
)

// StreamRule represents a stream rule.
type glStreamRule struct {
	ID          string `json:"id,omitempty"`
	StreamID    string `json:"stream_id,omitempty"`
	Field       string `json:"field,omitempty"`
	Value       string `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Type        int    `json:"type,omitempty"`
	Inverted    bool   `json:"inverted,omitempty"`
}

type glStream struct {
	Id                             string         `json:"id,omitempty"`
	Title                          string         `json:"title"`
	Description                    string         `json:"description"`
	Rules                          []glStreamRule `json:"rules"`
	RemoveMatchesFromDefaultStream bool           `json:"remove_matches_from_default_stream"`
	IndexSetID                     string         `json:"index_set_id"`
}

type glStreams struct {
	Streams []glStream `json:"streams"`
}

func ProvisionStream(ctx context.Context, log logr.Logger, data *GraylogProvisioningData) error {

	var (
		err    error
		client GraylogClient
	)

	client, err = CreateClient(log)
	if err != nil {
		return err
	}

	// check existing streams
	streams := &glStreams{}
	err = client.callAPIExpect(ctx, "GET", "/api/streams", nil, streams, 200)
	if err != nil {
		return err
	}

	for _, stream := range streams.Streams {
		if stream.Title == data.Name {
			log.Info("Stream already provisioned")
			data.Stream.id = stream.Id
			return nil
		}
	}

	// create the stream
	stream := glStream{
		Title:                          data.Name,
		Description:                    data.Name + "@" + OPERATOR_INFO,
		IndexSetID:                     data.IndexSet.id,
		RemoveMatchesFromDefaultStream: true,
		Rules: []glStreamRule{
			{
				Field: data.Stream.RuleFieldName,
				Value: data.Name,
				Type:  1,
			},
		},
	}

	response := struct {
		StreamId string `json:"stream_id"`
	}{}

	err = client.callAPIExpect(ctx, "POST", "/api/streams", stream, &response, 201)
	if err != nil {
		return err
	}

	data.Stream.id = response.StreamId
	log.Info("Stream created", "stream", stream)

	// start the stream
	err = client.callAPIExpect(ctx, "POST", "/api/streams/"+data.Stream.id+"/resume", nil, nil, 204)
	if err != nil {
		return err
	}

	log.Info("Stream started")

	/* By inspecting the Rest Calls from the Graylog UI, we see the following POST call executed:

	First there is a POST call to $GRAYLOG/api/authz/shares/entities/grn::::stream:60a242439e82ee1814ce2cd5/prepare, but it seems to be not
	mandatory, as we can create shares successfully with curl without `/prepare`.

	This is the call on "Save":
	curl "$GRAYLOG/api/authz/shares/entities/grn::::stream:60a242439e82ee1814ce2cd5" \
		  --data-raw '{"selected_grantee_capabilities":{"grn::::user:60a226a99e82ee1814ce0e92":"view"}}'

	60a242439e82ee1814ce2cd5 is the ID of the Stream
	60a226a99e82ee1814ce0e92 is the ID of the User

	*/

	grn := "grn::::stream:" + data.Stream.id

	share_request := struct {
		Selected_grantee_capabilities map[string]interface{} `json:"selected_grantee_capabilities,omitempty"`
	}{map[string]interface{}{
		"grn::::user:" + data.User.id: "view",
	}}

	err = client.callAPIExpect(ctx, "POST", "/api/authz/shares/entities/"+grn, share_request, nil, 200)
	if err != nil {
		return err
	}

	log.Info("Stream started")

	return nil
}
