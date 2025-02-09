package models

import "time"

type Owner struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"nodeId"`
	AvatarURL         string `json:"avatarUrl"`
	GravatarID        string `json:"gravatarId"`
	URL               string `json:"url"`
	HTMLURL           string `json:"htmlUrl"`
	FollowersURL      string `json:"followersUrl"`
	FollowingURL      string `json:"followingUrl"`
	GistsURL          string `json:"gistsUrl"`
	StarredURL        string `json:"starredUrl"`
	SubscriptionsURL  string `json:"subscriptionsUrl"`
	OrganizationsURL  string `json:"organizationsUrl"`
	ReposURL          string `json:"reposUrl"`
	EventsURL         string `json:"eventsUrl"`
	ReceivedEventsURL string `json:"receivedEventsUrl"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"siteAdmin"`
}

type Permissions map[string]string

type GithubAppResponse struct {
	ID            int64       `json:"id"`
	Slug          string      `json:"slug"`
	ClientID      string      `json:"clientId"`
	NodeID        string      `json:"nodeId"`
	Owner         Owner       `json:"owner"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	ExternalURL   string      `json:"externalUrl"`
	HTMLURL       string      `json:"htmlUrl"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
	Permissions   Permissions `json:"permissions"`
	Events        []string    `json:"events"`
	PEM           string      `json:"pem"`
	ClientSecret  string      `json:"clientSecret"`
	WebhookSecret string      `json:"webhookSecret"`
}

type GithubApp struct {
	ID            int64       `json:"id"`
	Slug          string      `json:"slug"`
	ClientID      string      `json:"clientId"`
	NodeID        string      `json:"nodeId"`
	Owner         Owner       `json:"owner"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	ExternalURL   string      `json:"externalUrl"`
	HTMLURL       string      `json:"htmlUrl"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
	Permissions   Permissions `json:"permissions"`
	Events        []string    `json:"events"`
	SourceID      string      `json:"sourceId"`
	ClientSecret  string      `json:"clientSecret"`
	WebhookSecret string      `json:"webhookSecret"`
	KeyID         string      `json:"keyId"`
}

func (r *GithubAppResponse) ToGithubApp() *GithubApp {
	return &GithubApp{
		ID:            r.ID,
		Slug:          r.Slug,
		ClientID:      r.ClientID,
		NodeID:        r.NodeID,
		Owner:         r.Owner,
		Name:          r.Name,
		Description:   r.Description,
		ExternalURL:   r.ExternalURL,
		HTMLURL:       r.HTMLURL,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		Permissions:   r.Permissions,
		Events:        r.Events,
		ClientSecret:  r.ClientSecret,
		WebhookSecret: r.WebhookSecret,
	}
}
