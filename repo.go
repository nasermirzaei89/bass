package bass

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"
)

type Resource interface {
	Name() string
}

type genericResource map[string]any

func (r genericResource) Name() string {
	return r["name"].(string) //nolint:forcetypeassert
}

type ResourceList struct {
	Items []Resource `json:"items"`
}

type ResourcesRepository interface {
	List(ctx context.Context) (list ResourceList, err error)
	Insert(ctx context.Context, item Resource) (err error)
	Get(ctx context.Context, itemName string) (item Resource, err error)
	Replace(ctx context.Context, item Resource) (err error)
	Delete(ctx context.Context, itemName string) (err error)
}

type ResourceExistsError struct {
	Name string
}

func (err ResourceExistsError) Error() string {
	return fmt.Sprintf("resource with name %q already exists", err.Name)
}

type ResourceNotFoundError struct {
	Name string
}

func (err ResourceNotFoundError) Error() string {
	return fmt.Sprintf("resource with name %q not found", err.Name)
}

type MemRepo struct {
	db map[string]Resource
	sync.Mutex
}

func (repo *MemRepo) List(_ context.Context) (ResourceList, error) {
	items := slices.Collect(maps.Values(repo.db))

	if items == nil {
		items = make([]Resource, 0)
	}

	res := ResourceList{
		Items: items,
	}

	return res, nil
}

func (repo *MemRepo) Insert(_ context.Context, item Resource) error {
	itemName := item.Name()

	if _, ok := repo.db[itemName]; ok {
		return ResourceExistsError{Name: itemName}
	}

	repo.Lock()
	defer repo.Unlock()

	repo.db[itemName] = item

	return nil
}

func (repo *MemRepo) Get(_ context.Context, itemName string) (Resource, error) {
	item, ok := repo.db[itemName]
	if !ok {
		return nil, ResourceNotFoundError{Name: itemName}
	}

	return item, nil
}

func (repo *MemRepo) Replace(_ context.Context, item Resource) error {
	itemName := item.Name()

	item, ok := repo.db[itemName]
	if !ok {
		return ResourceNotFoundError{Name: itemName}
	}

	repo.Lock()
	defer repo.Unlock()

	repo.db[itemName] = item

	return nil
}

func (repo *MemRepo) Delete(_ context.Context, itemName string) error {
	_, ok := repo.db[itemName]
	if !ok {
		return ResourceNotFoundError{Name: itemName}
	}

	repo.Lock()
	defer repo.Unlock()

	delete(repo.db, itemName)

	return nil
}

var _ ResourcesRepository = (*MemRepo)(nil)

func NewMemRepo() *MemRepo {
	return &MemRepo{
		db:    make(map[string]Resource),
		Mutex: sync.Mutex{},
	}
}
