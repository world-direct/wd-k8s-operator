package graylog

import (
	"context"

	"github.com/go-logr/logr"
)

type glIndexSetBasicInfo struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

type glIndexSetsBasicInfo struct {
	IndexSets []glIndexSetBasicInfo `json:"index_sets"`
	Total     int                   `json:"total"`
}

func ProvisionIndexSet(ctx context.Context, log logr.Logger, data *GraylogProvisioningData) error {

	var (
		err    error
		client GraylogClient
	)

	client, err = CreateClient(log)
	if err != nil {
		return err
	}

	// get all indexsets
	sets := &glIndexSetsBasicInfo{}
	err = client.callAPIExpect(ctx, "GET", "/api/system/indices/index_sets", nil, sets, 200)
	if err != nil {
		return err
	}

	// find our indexset
	var templateIndexSetId string
	for _, set := range sets.IndexSets {
		if set.Title == data.Name {
			data.IndexSet.ID = set.Id
			log.Info("Indexset already provisioned")
			return nil
		}

		if set.Title == data.IndexSet.TemplateName {
			templateIndexSetId = set.Id
		}
	}

	log.Info("Create new IndexSet by Template", "TemplateId", templateIndexSetId)

	// fetch raw json for the indexset
	indexSet := make(map[string]interface{})
	err = client.callAPIExpect(ctx, "GET", "/api/system/indices/index_sets/"+templateIndexSetId, nil, &indexSet, 200)
	if err != nil {
		return err
	}

	// overwrite fields needed to clone our indexset
	indexSet["id"] = nil
	indexSet["title"] = data.Name
	indexSet["description"] = data.Name + "@" + OPERATOR_INFO
	indexSet["index_prefix"] = data.Name + "-"

	// create the indexset
	err = client.callAPIExpect(ctx, "POST", "/api/system/indices/index_sets", indexSet, nil, 200)
	if err != nil {
		return err
	}

	return nil
}
