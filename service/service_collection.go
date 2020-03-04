package service

import (
	"github.com/TouchBistro/tb/util"
	"github.com/pkg/errors"
)

type ServiceCollection struct {
	sm  map[string][]Service
	len int
}

func NewServiceCollection() *ServiceCollection {
	return &ServiceCollection{sm: make(map[string][]Service)}
}

func (sc *ServiceCollection) Get(name string) (Service, error) {
	registryName, serviceName, err := util.SplitNameParts(name)
	if err != nil {
		return Service{}, errors.Wrapf(err, "invalid service name %s", name)
	}

	bucket, ok := sc.sm[serviceName]
	if !ok {
		return Service{}, errors.Errorf("No such service %s", serviceName)
	}

	// Handle shorthand syntax
	if registryName == "" {
		if len(bucket) > 1 {
			return Service{}, errors.Errorf("Muliple services named %s found", serviceName)
		}

		return bucket[0], nil
	}

	// Handle long syntax
	for _, s := range bucket {
		if s.RegistryName == registryName {
			return s, nil
		}
	}

	return Service{}, errors.Errorf("No such service %s", name)
}

func (sc *ServiceCollection) Set(value Service) error {
	if value.Name == "" || value.RegistryName == "" {
		return errors.Errorf("Name and RegistryName fields must not be empty to set Service")
	}

	fullName := value.FullName()
	registryName, serviceName, err := util.SplitNameParts(fullName)
	if err != nil {
		return errors.Wrapf(err, "invalid service name %s", fullName)
	}

	bucket, ok := sc.sm[serviceName]
	if !ok {
		sc.sm[serviceName] = []Service{value}
		sc.len++
		return nil
	}

	// Check for existing playlist to update
	for i, s := range bucket {
		if s.RegistryName == registryName {
			sc.sm[serviceName][i] = value
			return nil
		}
	}

	// No matching playlist found, add a new one
	sc.sm[serviceName] = append(bucket, value)
	sc.len++
	return nil
}

func (sc *ServiceCollection) Len() int {
	return sc.len
}

func (sc *ServiceCollection) ApplyOverrides(overrides map[string]ServiceOverride) error {
	for name, override := range overrides {
		s, err := sc.Get(name)
		if err != nil {
			return errors.Errorf("failed to get service %s to apply override to", name)
		}

		s, err = s.applyOverride(override)
		if err != nil {
			return errors.Errorf("failed to apply override to service %s", s.FullName())
		}

		// Name and RegistryName are validated before a service is added to the ServiceCollection
		// so it's safe to use them to directly update the service
		for i, el := range sc.sm[s.Name] {
			if el.RegistryName == s.RegistryName {
				sc.sm[s.Name][i] = s
				break
			}
		}
	}

	return nil
}

type iterator struct {
	sc   *ServiceCollection
	keys []string
	// Index of the current map key
	i int
	// Index of the current element in the bucket
	j int
}

// Iter creates a new iterator that can be used to iterate over the services in the ServiceCollection.
func (sc *ServiceCollection) Iter() *iterator {
	it := iterator{
		sc:   sc,
		keys: make([]string, len(sc.sm)),
	}

	i := 0
	for k := range sc.sm {
		it.keys[i] = k
		i++
	}

	return &it
}

// Next returns the next element in the ServiceCollection currently being iterated.
// NOTE: This does not check if a next element exists. It is the callers responsibilty
// to ensure there is a next element using the HasNext() method.
func (it *iterator) Next() Service {
	// Get value at current index
	key := it.keys[it.i]
	bucket := it.sc.sm[key]
	val := bucket[it.j]

	// Update indices
	if it.j == len(bucket)-1 {
		// Move to next bucket
		it.i++
		it.j = 0
	} else {
		it.j++
	}

	return val
}

// HasNext checks if the iterator has an next element and the Next() method can be safely called.
func (it *iterator) HasNext() bool {
	sm := it.sc.sm
	// Another element exists if i isn't on the last map key
	// and j isn't on the last bucket index
	return it.i < len(sm) && it.j < len(sm[it.keys[it.i]])
}
