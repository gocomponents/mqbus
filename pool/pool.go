package pool

import (
	"errors"
	"fmt"
	"sync"
)

var (
	ErrClosed = errors.New("pool is closed")
)

type Pool interface {
	Get() (Object, error)

	Close()

	Len() int
}

type Object interface {
	Dispose() error
}

type PooledObject struct {
	rwMu     sync.RWMutex
	Obj      Object
	unusable bool
	c        *channelPool
}

func (po *PooledObject) Dispose() error {
	po.rwMu.RLock()

	defer po.rwMu.RUnlock()

	if po.unusable {
		if po.Obj != nil {
			po.Obj.Dispose()
		}
		return nil
	}

	return po.c.put(po.Obj)
}

func (po *PooledObject) MarkUnusable() {
	po.rwMu.Lock()

	defer po.rwMu.Unlock()

	po.unusable = true
}

type ObjectFactory func() (Object, error)

type channelPool struct {
	mu      sync.RWMutex
	objects chan Object

	factory ObjectFactory
}

func NewChannelPool(initialCap, maxCap int, factory ObjectFactory) (Pool, error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid >maxCap")
	}

	c := &channelPool{
		objects: make(chan Object, maxCap),
		factory: factory,
	}

	for i := 0; i < initialCap; i++ {
		object, err := factory()
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("%s", err)
		}
		c.objects <- object
	}

	return c, nil
}

func (c *channelPool) wrapObject(object Object) Object {
	po := &PooledObject{c: c}
	po.Obj = object
	return po
}

func (c *channelPool) getObjectsAndFactory() (chan Object, ObjectFactory) {
	c.mu.RLock()
	objects := c.objects
	factory := c.factory
	c.mu.RUnlock()
	return objects, factory
}

func (c *channelPool) Get() (Object, error) {
	objects, _ := c.getObjectsAndFactory()
	if objects == nil {
		return nil, ErrClosed
	}

	select {
	case object := <-objects:
		if object == nil {
			return nil, ErrClosed
		}

		return c.wrapObject(object), nil
		//default:
		//	obj, err := factory()
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	return c.wrapObject(obj), nil
	}
}

func (c *channelPool) put(obj Object) error {
	if obj == nil {
		return errors.New("obj is nil. rejecting")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.objects == nil {
		return obj.Dispose()
	}

	select {
	case c.objects <- obj:
		return nil
	default:
		return obj.Dispose()
	}
}

func (c *channelPool) Close() {
	c.mu.Lock()
	objects := c.objects
	c.objects = nil
	c.factory = nil
	c.mu.Unlock()

	if objects == nil {
		return
	}

	close(objects)
	for obj := range objects {
		obj.Dispose()
	}
}

func (c *channelPool) Len() int {
	objects, _ := c.getObjectsAndFactory()
	return len(objects)
}