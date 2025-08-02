package bass

import "time"

type Metadata struct {
	UID          string    `json:"uid"`
	PackageName  string    `json:"packageName"`
	APIVersion   string    `json:"apiVersion"`
	ResourceType string    `json:"resourceType"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ListMetadata struct {
	PackageName  string `json:"packageName"`
	APIVersion   string `json:"apiVersion"`
	ResourceType string `json:"resourceType"`
}
