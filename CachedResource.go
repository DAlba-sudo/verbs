package verbs

import (
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sync"
)

type cachedResourceMetadata struct {
	// the time that the resource was cached (for time based
	// invalidation policies)
	CachedAt time.Time

	// the resource that is being cached
	Resource any
}

type cachedResource struct {
	InvalidationPolicy func(*http.Request, cachedResourceMetadata) bool
	AcquisitionPolicy  func(w http.ResponseWriter, r *http.Request, m map[string]any) (any, error)

	fingerprint []string
	cache       map[string]cachedResourceMetadata
	name        string
	concurrency chan struct{}
	cacheMutex  *sync.RWMutex
}

func (c *cachedResource) AddResource(hash string, v any) {
	c.cache[hash] = cachedResourceMetadata{
		CachedAt: time.Now(),
		Resource: v,
	}
}

type QueryCachedResourceOptions struct {
	// this array of strings will define the elements in the request's FormValue
	// that will be used to determine the cache key.
	Fingerprint []string

	// this represents the maximum number of allowed concurrent operations, managed
	// via a go channel.
	MaxConcurrentAcquisitions int

	// this function returns true if the cached resource should be invalidated and revmoed
	// from the cache.
	InvalidationPolicy func(*http.Request, cachedResourceMetadata) bool

	// this function will take the request data and return the resource that
	// should be cached.
	AcquisitionPolicy func(w http.ResponseWriter, r *http.Request, m map[string]any) (any, error)
}

// This is a request aware bridge which is able to cache resources based on
// input queries (i.e., name, search, etc).
func QueryCachedResource(name string, opts QueryCachedResourceOptions) cachedResource {
	if opts.MaxConcurrentAcquisitions <= 0 {
		opts.MaxConcurrentAcquisitions = 1
	}

	if opts.AcquisitionPolicy == nil {
		panic("AcquisitionPolicy must be provided in a CachedResource")
	}

	// default invalidation policy is to never invalidate the cache,
	// so we will set it as such
	if opts.InvalidationPolicy == nil {
		opts.InvalidationPolicy = func(*http.Request, cachedResourceMetadata) bool {
			return false
		}
	}

	return cachedResource{
		InvalidationPolicy: opts.InvalidationPolicy,
		AcquisitionPolicy:  opts.AcquisitionPolicy,

		fingerprint: opts.Fingerprint,
		name:        name,
		cache:       make(map[string]cachedResourceMetadata),
		concurrency: make(chan struct{}, opts.MaxConcurrentAcquisitions),
		cacheMutex:  &sync.RWMutex{},
	}
}

func (c cachedResource) Data(w http.ResponseWriter, r *http.Request, model map[string]any) (any, error) {
	// construt the hash key that will be used to acces the cache.
	hashable_key := strings.Builder{}
	for _, key := range c.fingerprint {
		value := r.FormValue(key)
		if value == "" {
			continue
		}

		hashable_key.WriteString(value)
	}

	md5_hash := fmt.Sprintf("%x", md5.Sum([]byte(hashable_key.String())))

	// check for the resource in the cache if it exists
	c.cacheMutex.RLock()
	resource, ok := c.cache[md5_hash]
	if ok {
		invalid := c.InvalidationPolicy(r, resource)
		if !invalid {
			logger.Info("cached resource hit", "hash", md5_hash)
			c.cacheMutex.RUnlock()
			return resource.Resource, nil
		}
		c.cacheMutex.RUnlock()

		c.cacheMutex.Lock()
		delete(c.cache, md5_hash)
		c.cacheMutex.Unlock()
	} else {
		c.cacheMutex.RUnlock()
	}

	select {
	case c.concurrency <- struct{}{}:
		c.cacheMutex.Lock()
		defer c.cacheMutex.Unlock()
		defer func() { <-c.concurrency }()

		res, err := c.AcquisitionPolicy(w, r, model)
		if err != nil {
			// the resource could not be retrieved, so we will return nil and the error.
			logger.Error("Failed to acquire resource", "error", err)
			return nil, err
		}

		// we acquired it successfully, so we will add it to the cache and return it.
		c.AddResource(md5_hash, res)
		logger.Info("cached resource replenished", "hash", md5_hash)
		return res, nil
	default:
		logger.Warn("Max concurrent acquisitions reached for cached resource", "name", c.name)
		return nil, errors.New("max concurrent acquisitions reached for cached resource")
	}
}

func (c cachedResource) Name() string {
	return c.name
}
