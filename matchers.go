package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type MatcherFunc func(root interface{}, expectedValue interface{}, path ...string) (bool, error)

func ChooseMatcher(path string) MatcherFunc {
	exactMatch := !strings.HasPrefix(path, "~")

	if exactMatch {
		return equalsByPath
	} else {
		return searchByPath
	}
}

func equalsByPath(m interface{}, expectedValue interface{}, path ...string) (bool, error) {

	val, err := getByPath(m, path...)
	return (expectedValue == val), err
}

// exact value by exact path
func getByPath(m interface{}, path ...string) (interface{}, error) {

	for _, p := range path {
		//fmt.Println(p)
		funcVal, ok := pathFunction(m, p)
		if ok {
			return funcVal, nil
		}

		idx, err := strconv.Atoi(p)
		if err != nil {
			//fmt.Println(err)
			mp, ok := m.(map[string]interface{})
			if !ok {
				str := fmt.Sprintf("Can't cast to Map and get key [%v] in path %v", p, path)
				return nil, errors.New(str)
			}
			if val, ok := mp[p]; ok {
				m = val
			} else {
				str := fmt.Sprintf("Map key [%v] does not exist in path %v", p, path)
				return nil, errors.New(str)
			}
		} else {
			arr, ok := m.([]interface{})
			if !ok {
				str := fmt.Sprintf("Can't cast to Array and get index [%v] in path %v", idx, path)
				return nil, errors.New(str)
			}
			if idx >= len(arr) {
				str := fmt.Sprintf("Array only has [%v] elements. Can't get element by index [%v] (counts from zero)", len(arr), idx)
				return nil, errors.New(str)
			}
			m = arr[idx]
		}
	}

	return m, nil
}

// search passing maps and arrays
func searchByPath(m interface{}, expectedValue interface{}, path ...string) (bool, error) {
	//fmt.Println("searchByPath", m, expectedValue, path, reflect.TypeOf(expectedValue))
	switch typedExpectedValue := expectedValue.(type) {
	case []interface{}:
		for _, obj := range typedExpectedValue {
			if ok, err := searchByPath(m, obj, path...); !ok {
				return false, err
			}
		}
		return true, nil
	case interface{}:
		for idx, p := range path {
			//fmt.Println("iter ", idx, p)
			if funcVal, ok := pathFunction(m, p); ok {
				if typedExpectedValue == funcVal {
					return true, nil
				}
			}

			switch typedM := m.(type) {
			case map[string]interface{}:
				m = typedM[p]
				//fmt.Println("mapped", m, reflect.TypeOf(m))

				switch typedM := m.(type) {
				case []interface{}:
					for _, v := range typedM {
						if v == typedExpectedValue {
							return true, nil
						}
					}
				}

				if m == typedExpectedValue {
					return true, nil
				}
			case []interface{}:
				//fmt.Println("arr ", path[idx:])
				for _, obj := range typedM {
					found, err := searchByPath(obj, typedExpectedValue, path[idx:]...)
					if found {
						return true, err
					}
				}
			}
		}
	}
	return false, nil
}

func pathFunction(m interface{}, pathPart string) (float64, bool) {

	if pathPart == "size()" {
		if arr, ok := m.([]interface{}); ok {
			return float64(len(arr)), true
		}
	}

	return -1, false
}
