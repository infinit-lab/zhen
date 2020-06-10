package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

type request struct {
	method string
	url string
	version string
	headers map[string][]string
	body []byte
}

func parse(cache string) (*request, error) {
	cache = checkMethod(cache)
	req := new(request)
	req.headers = make(map[string][]string)
	index := 0
	contentLength := 0
	for index != -1 {
		i := strings.Index(cache[index:], "\r\n")
		if i == -1 {
			return nil, errors.New("header is not finished")
		}
		if i == 0 {
			index += 2
			if contentLength == 0 {
				cache = cache[index:]
				if req.method == "" {
					return parse(cache)
				}
				return req, nil
			} else {
				if len(cache[index:]) < contentLength {
					return nil, errors.New("content is not finished")
				}
				req.body = []byte(cache[index:index+contentLength])
				cache = cache[index+contentLength:]
				return req, nil
			}
		}
		if index == 0 {
			sections := strings.Split(cache[index:index+i], " ")
			if len(sections) != 3 {
				cache = cache[index+i+2:]
				return parse(cache)
			}
			req.method = sections[0]
			req.url = sections[1]
			req.version = sections[2]
		} else {
			token := strings.Index(cache[index:index+i], ":")
			if token != -1 {
				key := strings.TrimSpace(cache[index:index+token])
				value := strings.TrimSpace(cache[index+token+1:index+i])
				values, ok := req.headers[key]
				if !ok {
					values = []string{}
				}
				values = append(values, value)
				req.headers[key] = values
				if strings.ToLower(key) == "content-length" {
					contentLength, _ = strconv.Atoi(value)
				}
			}
		}
		index += i + 2
	}
	return nil, nil
}

func checkMethod(cache string) string {
	methods := []string {
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodTrace,
		http.MethodOptions,
		http.MethodConnect,
	}
	for _, method := range methods {
		index := strings.Index(cache, method)
		if index != -1 {
			return cache[index:]
		}
	}
	return ""
}
