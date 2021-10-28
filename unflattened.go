package unflattened

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

type (
	Flattenable interface {
		UFUnlinkChildren() error
		UFGetChildren() ([]Flattenable, error)
	}

	Un interface {
		UFKey() string
		UFParentKey() string
		UFAppendChild(Un) error
	}

	UnFlattenable interface {
		Un
		Flattenable
	}

	Flattener func(interface{}) (UnFlattenable, error)
)

func Flatten(obj interface{}) ([]interface{}, error) {
	if objFlattenable, castable := obj.(Flattenable); castable {
		if flattenedUF, err := FlattenUF(objFlattenable); err == nil {
			flattened := make([]interface{}, 0, len(flattenedUF))
			for _, entityUF := range flattenedUF {
				flattened = append(flattened, entityUF)
			}

			return flattened, nil
		} else {
			return nil, err
		}
	}

	return nil, fmt.Errorf("unflattened.Flatten")
}

func FlattenUF(obj Flattenable) ([]Flattenable, error) {
	const BufferSize = 128

	entitiesCh := make(chan Flattenable, BufferSize)
	entities := make([]Flattenable, 0, BufferSize)

	errs, _ := errgroup.WithContext(context.Background())
	errs.Go(func() error {
		err := sendAppendSink(entitiesCh, obj)
		close(entitiesCh)
		return err
	})

	for got := range entitiesCh {
		entities = append(entities, got)
	}

	return entities, errs.Wait()
}

func sendAppendSink(sink chan<- Flattenable, entity Flattenable) error {
	sink <- entity
	if children, err := entity.UFGetChildren(); err != nil {
		return err
	} else {
		for _, chEntity := range children {
			sendAppendSink(sink, chEntity)
		}

		if err := entity.UFUnlinkChildren(); err != nil {
			return err
		}
	}

	return nil
}

func UnflattenMapUF(entities map[string]UnFlattenable) ([]UnFlattenable, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("unflattened.UnflattenMap: got empty input map")
	}

	roots := make([]UnFlattenable, 0, len(entities)/2)

	for _, entity := range entities {
		if parent, isNotRoot := entities[entity.UFParentKey()]; isNotRoot {
			if err := parent.UFAppendChild(entity); err != nil {
				return nil, err
			}
		} else {
			roots = append(roots, entity)
		}
	}

	return roots, nil
}

func Unflatten(entities []interface{}) ([]UnFlattenable, error) {
	entitiesUF := make([]UnFlattenable, 0, len(entities))
	for _, entity := range entities {
		if entityUF, castable := entity.(UnFlattenable); castable {
			entitiesUF = append(entitiesUF, entityUF)
		} else {
			return nil, fmt.Errorf("non castable")
		}
	}

	return UnflattenUF(entitiesUF)
}

func UnflattenUF(entities []UnFlattenable) ([]UnFlattenable, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("unflattened.Unflatten: got empty input slice")
	}

	entitiesMap := make(map[string]UnFlattenable, len(entities))
	for _, entity := range entities {
		entitiesMap[entity.UFKey()] = entity
	}

	return UnflattenMapUF(entitiesMap)
}
