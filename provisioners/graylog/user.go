package graylog

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

// User represents a user in the Graylog API
type glUser struct {

	// a unique user name used to log in with.
	// ex. "local:admin"
	Username string `json:"username,omitempty"`
	// the contact email address
	Email string `json:"email,omitempty"`
	// a descriptive name for this account, e.g. the full name.
	FullName string `json:"full_name,omitempty"`
	Password string `json:"password,omitempty"`

	ID string `json:"id,omitempty"`

	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions"`
}

func (client GraylogClient) tryGetUserByName(ctx context.Context, username string) (*glUser, error) {
	user := &glUser{}
	sc, err := client.callAPI(ctx, "GET", "/api/users/"+username, nil, user)

	switch sc {
	case 200:
		return user, nil
	case 404:
		return nil, nil
	default:
		return nil, err
	}
}

func ProvisionUser(ctx context.Context, log logr.Logger, data *GraylogProvisioningData) error {

	var (
		err error
	)

	log.Info("Start provisioning User", "GraylogUser", data.Name)

	client, err := CreateClient(log)
	if err != nil {
		return err
	}

	// check user existance
	user, err := client.tryGetUserByName(ctx, data.Name)
	if user != nil {
		data.User.id = user.ID
		log.Info("User already provisioned")
		return nil
	} else if err != nil {
		return err
	}

	// initialize the user
	user = &glUser{
		Username:    data.Name,
		FullName:    data.Name,
		Password:    data.User.InitialPassword,
		Email:       data.Name + "@" + OPERATOR_INFO,
		Roles:       data.User.Roles,
		Permissions: []string{},
	}

	// create the user
	err = client.callAPIExpect(ctx, "POST", "/api/users", user, nil, 201)
	if err != nil {
		return errors.Wrapf(err, "Error creating user '%s'", data.Name)
	}

	log.Info("User created")

	// No body is returned by the POST /api/users, so we need to read the ID with a new request
	user, err = client.tryGetUserByName(ctx, data.Name)
	if err != nil {
		return err
	}

	data.User.id = user.ID

	return nil
}
