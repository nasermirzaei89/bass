package bass

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
)

type Resource struct {
	Metadata   Metadata       `json:"metadata"`
	Properties map[string]any `json:",inline"`
}

type ResourceList struct {
	Metadata ListMetadata `json:"metadata"`
	Items    []*Resource  `json:"items"`
}

type ResourcesRepository interface {
	List(ctx context.Context, packageName, apiVersion, resourceType string) (list ResourceList, err error)
	Create(ctx context.Context, item *Resource) (err error)
	Get(ctx context.Context, packageName, resourceType, name string) (item *Resource, err error)
	Update(ctx context.Context, item *Resource) (err error)
	Delete(ctx context.Context, packageName, resourceTypePlural, name string) (err error)
}

type ResourceExistsError struct {
	PackageName  string
	ResourceType string
	Name         string
}

func (err ResourceExistsError) Error() string {
	return fmt.Sprintf("resource with name %q and resource type %q and package %q already exists", err.Name, err.ResourceType, err.PackageName)
}

type ResourceNotFoundError struct {
	PackageName  string
	ResourceType string
	Name         string
}

func (err ResourceNotFoundError) Error() string {
	return fmt.Sprintf("resource with name %q and resource type %q and package %q not found", err.Name, err.ResourceType, err.PackageName)
}

type MemRepo struct {
	sync.Mutex

	db map[string]*Resource
}

var _ ResourcesRepository = (*MemRepo)(nil)

func NewMemRepo() *MemRepo {
	return &MemRepo{
		db:    make(map[string]*Resource),
		Mutex: sync.Mutex{},
	}
}

func (repo *MemRepo) List(_ context.Context, packageName, apiVersion, resourceType string) (ResourceList, error) {
	allItems := slices.Collect(maps.Values(repo.db))

	resourceItems := slices.Collect(func(yield func(*Resource) bool) {
		for _, item := range allItems {
			if item.Metadata.ResourceType == resourceType && item.Metadata.PackageName == packageName {
				yield(item)
			}
		}
	})

	if resourceItems == nil {
		resourceItems = make([]*Resource, 0)
	}

	res := ResourceList{
		Metadata: ListMetadata{
			PackageName:  packageName,
			APIVersion:   apiVersion,
			ResourceType: resourceType + "List",
		},
		Items: resourceItems,
	}

	return res, nil
}

func (repo *MemRepo) Create(_ context.Context, item *Resource) error {
	_, ok := repo.get(item.Metadata.PackageName, item.Metadata.ResourceType, item.Metadata.Name)
	if ok {
		return ResourceExistsError{
			PackageName:  item.Metadata.PackageName,
			ResourceType: item.Metadata.ResourceType,
			Name:         item.Metadata.Name,
		}
	}

	repo.put(item)

	return nil
}

func (repo *MemRepo) Get(_ context.Context, packageName, resourceType, name string) (*Resource, error) {
	item, ok := repo.get(packageName, resourceType, name)
	if !ok {
		return nil, ResourceNotFoundError{
			PackageName:  packageName,
			ResourceType: resourceType,
			Name:         name,
		}
	}

	return item, nil
}

func (repo *MemRepo) Update(_ context.Context, item *Resource) error {
	_, ok := repo.get(item.Metadata.PackageName, item.Metadata.ResourceType, item.Metadata.Name)
	if !ok {
		return ResourceNotFoundError{
			PackageName:  item.Metadata.PackageName,
			ResourceType: item.Metadata.ResourceType,
			Name:         item.Metadata.Name,
		}
	}

	repo.put(item)

	return nil
}

func (repo *MemRepo) Delete(_ context.Context, packageName, resourceType, name string) error {
	_, ok := repo.get(packageName, resourceType, name)
	if !ok {
		return ResourceNotFoundError{
			PackageName:  packageName,
			ResourceType: resourceType,
			Name:         name,
		}
	}

	repo.delete(packageName, resourceType, name)

	return nil
}

func (repo *MemRepo) put(item *Resource) {
	repo.Lock()
	defer repo.Unlock()

	key := item.Metadata.PackageName + "/" + item.Metadata.ResourceType + "/" + item.Metadata.Name

	repo.db[key] = item
}

func (repo *MemRepo) delete(packageName, resourceType, name string) {
	repo.Lock()
	defer repo.Unlock()

	key := packageName + "/" + resourceType + "/" + name

	delete(repo.db, key)
}

func (repo *MemRepo) get(packageName, resourceType, name string) (*Resource, bool) {
	key := packageName + "/" + resourceType + "/" + name

	item, ok := repo.db[key]

	return item, ok
}
