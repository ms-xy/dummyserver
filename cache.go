package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	__cache_path__ string
)

func init() {
	if cwd, err := os.Getwd(); err != nil {
		panic(err)
	} else {
		__cache_path__ = filepath.Join(cwd, ".cache")
	}
}

type Cache interface {
	Add(key string, value any, timeout time.Duration)
	Get(key string, defaultValue any) any
	ToMap() map[string]any
	Clear()
}

func NewCache(additionalRemovalAction func(key string, value any)) Cache {
	return &cacheImpl{
		internalMap: make(map[string]any),
		timersLock: &sync.Mutex{},
		timers: make(map[string]*time.Timer),
		additionalRemovalAction: additionalRemovalAction,
	}
}

type cacheImpl struct {
	internalMap map[string]any
	timersLock *sync.Mutex
	timers map[string]*time.Timer
	additionalRemovalAction func(key string, value any)
}

func (cache *cacheImpl) resetRemovalTimer(key string, timeout time.Duration) {
	cache.timersLock.Lock()
	defer cache.timersLock.Unlock()
	timer, exists := cache.timers[key]
	if exists {
		timer.Reset(timeout)
	} else {
		timer := time.AfterFunc(timeout, func() {
			cache.timersLock.Lock()
			defer cache.timersLock.Unlock()
			if _, exists := cache.timers[key]; !exists {
				return
			}
			value := cache.internalMap[key]
			delete(cache.timers, key)
			delete(cache.internalMap, key)
			cache.additionalRemovalAction(key, value)
		})
		cache.timers[key] = timer
	}
}

func (cache *cacheImpl) Add(key string, value any, timeout time.Duration) {
	cache.internalMap[key] = value
	if timeout > 0 {
		cache.resetRemovalTimer(key, timeout)
	}
}

func (cache *cacheImpl) Get(key string, defaultValue any) any {
	if value, exists := cache.internalMap[key]; exists {
		return value
	}
	return defaultValue
}

func (cache *cacheImpl) ToMap() map[string]any {
	return cache.internalMap
}

func (cache *cacheImpl) Clear() {
	cache.timersLock.Lock()
	defer cache.timersLock.Unlock()
	for key, value := range cache.internalMap {
		delete(cache.internalMap, key)
		delete(cache.timers, key)
		cache.additionalRemovalAction(key, value)
	}
}

type CacheFile interface {
	Remove() error
	AddHeaders(response http.ResponseWriter)
	Copy(response http.ResponseWriter) error
}

type cacheFileImpl struct {
	tmpFile *os.File
	Name string
	Headers map[string][]string
}

func NewCacheFile(file multipart.File, header *multipart.FileHeader) (cacheFile *cacheFileImpl, cacheErr error) {
	cacheFile = &cacheFileImpl{
		Headers: header.Header,
	}
	defer func() {
		if r := recover(); r != nil {
			if cacheFile.tmpFile != nil {
				if err := os.Remove(cacheFile.tmpFile.Name()); err != nil {
					log.Printf("[ERROR] Failed to remove tmp file: %v", err)
				}
			}
			if err, castOk := r.(error); castOk {
				cacheErr = err
			} else {
				cacheErr = fmt.Errorf("%v", r)
			}
		}
	}()
	if finfo, err := os.Stat(__cache_path__); errors.Is(err, fs.ErrNotExist)  {
		if err := os.Mkdir(__cache_path__, 0700); err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	} else if !finfo.IsDir() {
		panic(fmt.Errorf("'%s' exists, but not a directory", __cache_path__))
	}
	tmpFile, err := os.CreateTemp(__cache_path__, "dummyserver_cachefile_*")
	if err != nil {
		panic(err)
	}
	cacheFile.tmpFile = tmpFile
	if _, err := io.Copy(cacheFile.tmpFile, file); err != nil {
		panic(err)
	}
	cacheFile.tmpFile.Sync()
	return
}

func (cacheFile *cacheFileImpl) Remove() error {
	path := cacheFile.tmpFile.Name()
	cacheFile.tmpFile.Close()
	return os.Remove(path)
}

func (cacheFile *cacheFileImpl) AddHeaders(response http.ResponseWriter) {
	for key, value := range cacheFile.Headers {
		response.Header().Add(key, strings.Join(value, "; "))
	}
}

func (cacheFile *cacheFileImpl) Copy(response http.ResponseWriter) error {
	if file, err := os.Open(cacheFile.tmpFile.Name()); err != nil {
		return err
	} else {
		_, err = io.Copy(response, file)
		return err
	}
}
