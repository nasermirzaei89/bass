package bass

import (
	"context"
	"fmt"
)

type ResourceTypeDefinition struct {
	Metadata     Metadata                        `json:"metadata"`
	Package      string                          `json:"package"`
	Versions     []ResourceTypeDefinitionVersion `json:"versions"`
	ResourceType string                          `json:"resourceType"`
	Plural       string                          `json:"plural"`
}

type ResourceTypeDefinitionVersion struct {
	Name   string         `json:"name"`
	Schema map[string]any `json:"schema"`
}

type ResourceTypeDefinitionNotFoundError struct {
	PackageName        string
	ResourceTypePlural string
}

func (err ResourceTypeDefinitionNotFoundError) Error() string {
	return fmt.Sprintf("resource type definition not found for package %q and resource type %q", err.PackageName, err.ResourceTypePlural)
}

func (h *Handler) getResourceTypeDefinition(ctx context.Context, packageName, resourceTypePlural string) (*ResourceTypeDefinition, error) {
	name := resourceTypePlural + "." + packageName

	if packageName == "core" {
		resourceTypeDefinition, err := h.getCoreResourceTypeDefinition(ctx, resourceTypePlural)
		if err != nil {
			return nil, fmt.Errorf("failed to get core resource type definition: %w", err)
		}
		return resourceTypeDefinition, nil
	}

	item, err := h.repo.Get(ctx, "core", "ResourceTypeDefinition", name)
	if err != nil {
		return nil, ResourceTypeDefinitionNotFoundError{
			PackageName:        packageName,
			ResourceTypePlural: resourceTypePlural,
		}
	}

	resourceTypeDefinition := &ResourceTypeDefinition{
		Metadata:     item.Metadata,
		Package:      item.Properties["package"].(string),
		ResourceType: item.Properties["resourceType"].(string),
		Plural:       item.Properties["plural"].(string),
	}
	versions, ok := item.Properties["versions"].([]any)
	if !ok {
		return nil, fmt.Errorf("resource type %q has invalid versions property", name)
	}

	for _, version := range versions {
		versionMap, ok := version.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("resource type %q has invalid version property", name)
		}
		schema, ok := versionMap["schema"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("resource type %q has invalid schema property in version", name)
		}
		resourceTypeDefinition.Versions = append(resourceTypeDefinition.Versions, ResourceTypeDefinitionVersion{
			Schema: schema,
		})
	}

	return resourceTypeDefinition, nil
}

func (h *Handler) getCoreResourceTypeDefinition(ctx context.Context, resourceTypePlural string) (*ResourceTypeDefinition, error) {
	switch resourceTypePlural {
	case "resourcetypedefinitions":
		return &ResourceTypeDefinition{
			Metadata: Metadata{
				PackageName:  "core",
				APIVersion:   "v1",
				ResourceType: "ResourceTypeDefinition",
				Name:         "ResourceTypeDefinition.core",
			},
			Package:      "core",
			ResourceType: "ResourceTypeDefinition",
			Plural:       "ResourceTypeDefinitions",
			Versions: []ResourceTypeDefinitionVersion{
				{
					Name: "v1",
					Schema: map[string]any{
						"type":       "object",
						"properties": map[string]any{},
					},
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown core resource type %q", resourceTypePlural)
	}
}
