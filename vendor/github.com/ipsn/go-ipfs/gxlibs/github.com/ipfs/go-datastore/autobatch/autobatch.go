// Package autobatch provides a go-datastore implementation that
// automatically batches together writes by holding puts in memory until
// a certain threshold is met.
package autobatch

import (
	ds "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-datastore"
	dsq "github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-datastore/query"
)

// Datastore implements a go-datatsore.
type Datastore struct {
	child ds.Batching

	// TODO: discuss making ds.Batch implement the full ds.Datastore interface
	buffer           map[ds.Key][]byte
	maxBufferEntries int
}

// NewAutoBatching returns a new datastore that automatically
// batches writes using the given Batching datastore. The size
// of the memory pool is given by size.
func NewAutoBatching(d ds.Batching, size int) *Datastore {
	return &Datastore{
		child:            d,
		buffer:           make(map[ds.Key][]byte),
		maxBufferEntries: size,
	}
}

// Delete deletes a key/value
func (d *Datastore) Delete(k ds.Key) error {
	_, found := d.buffer[k]
	delete(d.buffer, k)

	err := d.child.Delete(k)
	if found && err == ds.ErrNotFound {
		return nil
	}
	return err
}

// Get retrieves a value given a key.
func (d *Datastore) Get(k ds.Key) ([]byte, error) {
	val, ok := d.buffer[k]
	if ok {
		return val, nil
	}

	return d.child.Get(k)
}

// Put stores a key/value.
func (d *Datastore) Put(k ds.Key, val []byte) error {
	d.buffer[k] = val
	if len(d.buffer) > d.maxBufferEntries {
		return d.Flush()
	}
	return nil
}

// Flush flushes the current batch to the underlying datastore.
func (d *Datastore) Flush() error {
	b, err := d.child.Batch()
	if err != nil {
		return err
	}

	for k, v := range d.buffer {
		err := b.Put(k, v)
		if err != nil {
			return err
		}
	}
	// clear out buffer
	d.buffer = make(map[ds.Key][]byte)

	return b.Commit()
}

// Has checks if a key is stored.
func (d *Datastore) Has(k ds.Key) (bool, error) {
	_, ok := d.buffer[k]
	if ok {
		return true, nil
	}

	return d.child.Has(k)
}

// GetSize implements Datastore.GetSize
func (d *Datastore) GetSize(k ds.Key) (int, error) {
	v, ok := d.buffer[k]
	if ok {
		return len(v), nil
	}

	return d.child.GetSize(k)
}

// Query performs a query
func (d *Datastore) Query(q dsq.Query) (dsq.Results, error) {
	err := d.Flush()
	if err != nil {
		return nil, err
	}

	return d.child.Query(q)
}

// DiskUsage implements the PersistentDatastore interface.
func (d *Datastore) DiskUsage() (uint64, error) {
	return ds.DiskUsage(d.child)
}
