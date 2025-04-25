package utils

import (
	"embed"
	"encoding/json"
	"time"
)

//go:embed items.json
var itemsFS embed.FS

type Category struct {
	Id            int       `json:"id"`
	Section       string    `json:"section"`
	Name          string    `json:"name"`
	IsGameRelated bool      `json:"is_game_related"`
	IsMining      bool      `json:"is_mining"`
	DateAdded     time.Time `json:"date_added"`
	DateModified  time.Time `json:"date_modified"`
}

type Item struct {
	Id                    int       `json:"id"`
	IdParent              int       `json:"id_parent"`
	IdCategory            int       `json:"id_category"`
	IdCompany             int       `json:"id_company"`
	IdVehicle             int       `json:"id_vehicle"`
	Name                  string    `json:"name"`
	Section               string    `json:"section"`
	Category              string    `json:"category"`
	CompanyName           string    `json:"company_name"`
	VehicleName           string    `json:"vehicle_name"`
	Slug                  string    `json:"slug"`
	Size                  string    `json:"size"`
	Uuid                  string    `json:"uuid"`
	UrlStore              string    `json:"url_store"`
	IsExclusivePledge     bool      `json:"is_exclusive_pledge"`
	IsExclusiveSubscriber bool      `json:"is_exclusive_subscriber"`
	IsExclusiveConcierge  bool      `json:"is_exclusive_concierge"`
	Screenshot            string    `json:"screenshot"`
	Attributes            string    `json:"attributes"`
	Notification          string    `json:"notification"`
	GameVersion           string    `json:"game_version"`
	DateAdded             time.Time `json:"date_added"`
	DateModified          time.Time `json:"date_modified"`
}

var items = []Item{}

func init() {
	// Read the items.json file
	data, err := itemsFS.ReadFile("items.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON data into a slice of Item structs
	if err := json.Unmarshal(data, &items); err != nil {
		panic(err)
	}
}

func GetItemNames() []string {
	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Name
	}

	return names
}
