package models

import "time"

type owner struct {
	Login             string `json:"login" db:"login"`
	ID                int    `json:"id" db:"id"`
	NodeID            string `json:"nodeId" db:"node_id"`
	AvatarURL         string `json:"avatarUrl" db:"avatar_url"`
	GravatarID        string `json:"gravatarId" db:"gravatar_id"`
	URL               string `json:"url" db:"url"`
	HTMLURL           string `json:"htmlUrl" db:"html_url"`
	FollowersURL      string `json:"followersUrl" db:"followers_url"`
	FollowingURL      string `json:"followingUrl" db:"following_url"`
	GistsURL          string `json:"gistsUrl" db:"gists_url"`
	StarredURL        string `json:"starredUrl" db:"starred_url"`
	SubscriptionsURL  string `json:"subscriptionsUrl" db:"subscriptions_url"`
	OrganizationsURL  string `json:"organizationsUrl" db:"organizations_url"`
	ReposURL          string `json:"reposUrl" db:"repos_url"`
	EventsURL         string `json:"eventsUrl" db:"events_url"`
	ReceivedEventsURL string `json:"receivedEventsUrl" db:"received_events_url"`
	Type              string `json:"type" db:"type"`
	SiteAdmin         bool   `json:"siteAdmin" db:"site_admin"`
}

type permissions map[string]string

type GithubAppResponse struct {
	ID            int64       `json:"id" db:"id"`
	Slug          string      `json:"slug" db:"slug"`
	ClientID      string      `json:"clientId" db:"client_id"`
	NodeID        string      `json:"nodeId" db:"node_id"`
	Owner         owner       `json:"owner" db:"owner"`
	Name          string      `json:"name" db:"name"`
	Description   string      `json:"description" db:"description"`
	ExternalURL   string      `json:"externalUrl" db:"external_url"`
	HTMLURL       string      `json:"htmlUrl" db:"html_url"`
	CreatedAt     time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time   `json:"updatedAt" db:"updated_at"`
	Permissions   permissions `json:"permissions" db:"permissions"`
	Events        []string    `json:"events" db:"events"`
	PEM           string      `json:"pem" db:"pem"`
	ClientSecret  string      `json:"clientSecret" db:"client_secret"`
	WebhookSecret string      `json:"webhookSecret" db:"webhook_secret"`
}

type GithubApp struct {
	ID            int64       `json:"id" db:"id"`
	Slug          string      `json:"slug" db:"slug"`
	ClientID      string      `json:"clientId" db:"client_id"`
	NodeID        string      `json:"nodeId" db:"node_id"`
	Owner         owner       `json:"owner" db:"owner"`
	Name          string      `json:"name" db:"name"`
	Description   string      `json:"description" db:"description"`
	ExternalURL   string      `json:"externalUrl" db:"external_url"`
	HTMLURL       string      `json:"htmlUrl" db:"html_url"`
	CreatedAt     time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time   `json:"updatedAt" db:"updated_at"`
	Permissions   permissions `json:"permissions" db:"permissions"`
	Events        []string    `json:"events" db:"events"`
	SourceID      string      `json:"sourceId" db:"source_id"`
	ClientSecret  string      `json:"clientSecret" db:"client_secret"`
	WebhookSecret string      `json:"webhookSecret" db:"webhook_secret"`
	KeyID         string      `json:"keyId" db:"key_id"`
}
