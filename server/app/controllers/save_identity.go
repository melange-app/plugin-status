package controllers

import (
	"fmt"
	"io"
	"melange/app/framework"
	"melange/app/models"
	"melange/app/packaging"
	"melange/dap"
	"melange/router"
	"net/http"

	"airdispat.ch/identity"

	gdb "github.com/huntaub/go-db"
)

// Profile is a JSONObject specifying a request to create a new identity
// (or profile, I suppose).
type Profile struct {
	// Profile Information
	FirstName string `json:"first"`
	LastName  string `json:"last"`
	About     string `json:"about"`
	Password  string `json:"password"`
	// AD Information
	Server  string `json:"server"`
	Tracker string `json:"tracker"`
	Alias   string `json:"alias"`
	// Profile Nickname
	Nickname string `json:"nickname"`
}

// SaveIdentity will create, register, alias, and save a new Identity.
type SaveIdentity struct {
	Tables   map[string]gdb.Table
	Store    *models.Store
	Packager *packaging.Packager
}

// Handle performs the specified functions.
func (i *SaveIdentity) Handle(req *http.Request) framework.View {
	// Decode Body
	profileRequest := &Profile{}
	err := DecodeJSONBody(req, &profileRequest)

	if err != nil && err != io.EOF {
		fmt.Println("Error occured while decoding body:", err)
		return framework.Error500
	}

	// Create Identity
	id, err := identity.CreateIdentity()
	if err != nil {
		fmt.Println("Error occured creating an identity:", err)
		return framework.Error500
	}

	//
	// Server Registration
	//

	// Extract Keys
	server, err := i.Packager.ServerFromId(profileRequest.Server)
	if err != nil {
		fmt.Println("Error occured getting server:", err)
		return &framework.HTTPError{
			ErrorCode: 500,
			Message:   "Couldn't get server for id.",
		}
	}

	// Run Registration
	client := &dap.Client{
		Key:    id,
		Server: server.Key,
	}
	err = client.Register(map[string][]byte{
		"name": []byte(profileRequest.FirstName + " " + profileRequest.LastName),
	})
	if err != nil {
		fmt.Println("Error occurred registering on Server", err)
		return framework.Error500
	}

	//
	// Tracker Registration
	//

	tracker, err := i.Packager.TrackerFromId(profileRequest.Tracker)
	if err != nil {
		fmt.Println("Error occured getting tracker:", err)
		return &framework.HTTPError{
			ErrorCode: 500,
			Message:   "Couldn't get tracker for id.",
		}
	}

	err = (&router.Router{
		Origin: id,
		TrackerList: []string{
			tracker.URL,
		},
	}).Register(id, profileRequest.Alias)

	if err != nil {
		fmt.Println("Error occurred registering on Tracker", err)
		return framework.Error500
	}

	//
	// Database Registration
	//

	modelID, err := models.CreateIdentityFromDispatch(id, "")
	if err != nil {
		fmt.Println("Error occurred encoding Id", err)
		return framework.Error500
	}

	modelID.Nickname = profileRequest.Nickname

	// Load Server Information
	modelID.Server = server.URL
	modelID.ServerKey = server.EncryptionKey
	modelID.ServerFingerprint = server.Fingerprint

	_, err = i.Tables["identity"].Insert(modelID).Exec(i.Store)
	if err != nil {
		fmt.Println("Error saving Identity", err)
		return framework.Error500
	}

	modelAlias := &models.Alias{
		Identity: gdb.ForeignKey(modelID),
		Location: tracker.URL,
		Username: profileRequest.Alias,
	}

	_, err = i.Tables["alias"].Insert(modelAlias).Exec(i.Store)
	if err != nil {
		fmt.Println("Error saving Alias", err)
		return framework.Error500
	}

	// Save as the current identity.
	err = i.Store.Set("current_identity", id.Address.String())
	if err != nil {
		fmt.Println("Error storing current_identity.", err)
		return framework.Error500
	}

	return &framework.JSONView{
		Content: map[string]interface{}{
			"error": false,
		},
	}
}