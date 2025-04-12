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
	Kind() string
}

type genericResource map[string]any

func (r genericResource) Name() string {
	return r["name"].(string) //nolint:forcetypeassert
}

func (r genericResource) Kind() string {
	return r["kind"].(string) //nolint:forcetypeassert
}

type ResourceList struct {
	Kind  string     `json:"kind"`
	Items []Resource `json:"items"`
}

type ResourcesRepository interface {
	List(ctx context.Context, itemKind string) (list ResourceList, err error)
	Insert(ctx context.Context, item Resource) (err error)
	Get(ctx context.Context, itemKind, itemName string) (item Resource, err error)
	Replace(ctx context.Context, item Resource) (err error)
	Delete(ctx context.Context, itemKind, itemName string) (err error)
}

type ResourceExistsError struct {
	Kind string
	Name string
}

func (err ResourceExistsError) Error() string {
	return fmt.Sprintf("resource with name %q and kind %q already exists", err.Name, err.Kind)
}

type ResourceNotFoundError struct {
	Kind string
	Name string
}

func (err ResourceNotFoundError) Error() string {
	return fmt.Sprintf("resource with name %q and kind %q not found", err.Name, err.Kind)
}

type MemRepo struct {
	db map[string]Resource
	sync.Mutex
}

func (repo *MemRepo) List(_ context.Context, itemKind string) (ResourceList, error) {
	allItems := slices.Collect(maps.Values(repo.db))

	kindItems := slices.Collect(func(yield func(Resource) bool) {
		for _, item := range allItems {
			if item.Kind() == itemKind {
				yield(item)
			}
		}
	})

	if kindItems == nil {
		kindItems = make([]Resource, 0)
	}

	res := ResourceList{
		Kind:  itemKind + "List",
		Items: kindItems,
	}

	return res, nil
}

func (repo *MemRepo) put(item Resource) {
	repo.Lock()
	defer repo.Unlock()

	repo.db[item.Kind()+"/"+item.Name()] = item
}

func (repo *MemRepo) delete(itemKind string, itemName string) {
	repo.Lock()
	defer repo.Unlock()

	delete(repo.db, itemKind+"/"+itemName)
}

func (repo *MemRepo) get(itemKind string, itemName string) (Resource, bool) {
	id := itemKind + "/" + itemName

	item, ok := repo.db[id]

	return item, ok
}

func (repo *MemRepo) Insert(_ context.Context, item Resource) error {
	itemName := item.Name()
	itemKind := item.Kind()

	_, ok := repo.get(itemKind, itemName)
	if ok {
		return ResourceExistsError{
			Kind: itemKind,
			Name: itemName,
		}
	}

	repo.put(item)

	return nil
}

func (repo *MemRepo) Get(_ context.Context, itemKind, itemName string) (Resource, error) {
	item, ok := repo.get(itemKind, itemName)
	if !ok {
		return nil, ResourceNotFoundError{
			Kind: itemKind,
			Name: itemName,
		}
	}

	return item, nil
}

func (repo *MemRepo) Replace(_ context.Context, item Resource) error {
	itemName := item.Name()
	itemKind := item.Kind()

	_, ok := repo.get(itemKind, itemName)
	if !ok {
		return ResourceNotFoundError{
			Kind: itemKind,
			Name: itemName,
		}
	}

	repo.put(item)

	return nil
}

func (repo *MemRepo) Delete(_ context.Context, itemKind, itemName string) error {
	_, ok := repo.get(itemKind, itemName)
	if !ok {
		return ResourceNotFoundError{
			Kind: itemKind,
			Name: itemName,
		}
	}

	repo.delete(itemKind, itemName)

	return nil
}

var _ ResourcesRepository = (*MemRepo)(nil)

func NewMemRepo() *MemRepo {
	return &MemRepo{
		db:    make(map[string]Resource),
		Mutex: sync.Mutex{},
	}
}
