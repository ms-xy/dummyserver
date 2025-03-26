package main

import (
	"errors"
	"strings"
)

type PathAccessor struct {
	config map[string]interface{}
}

func (pathAccessor *PathAccessor) Get(path string, defaultValue interface{}) interface{} {
	var getRecursive func(map[string]interface{}, []string) interface{}
	getRecursive = func(_map map[string]interface{}, _parts []string) interface{} {
		if _value, _ok := _map[_parts[0]]; !_ok {
			return defaultValue
		} else if len(_parts) == 1 {
			return _value
		} else if _actMap, _ok := _value.(map[string]interface{}); !_ok {
			return defaultValue
		} else {
			return getRecursive(_actMap, _parts[1:])
		}
	}
	return getRecursive(pathAccessor.config, strings.Split(path, "."))
}

var ErrPathNotFound = errors.New("path not found")

func (pathAccessor *PathAccessor) Must(path string) (interface{}, error) {
	var getRecursive func(map[string]interface{}, []string) (interface{}, error)
	getRecursive = func(_map map[string]interface{}, _parts []string) (interface{}, error) {
		if _value, _ok := _map[_parts[0]]; !_ok {
			return nil, ErrPathNotFound
		} else if len(_parts) == 1 {
			return _value, nil
		} else if _actMap, _ok := _value.(map[string]interface{}); !_ok {
			return nil, ErrPathNotFound
		} else {
			return getRecursive(_actMap, _parts[1:])
		}
	}
	return getRecursive(pathAccessor.config, strings.Split(path, "."))
}
