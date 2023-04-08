package shttp

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestMetricLabels(t *testing.T) {
	var req *http.Request
	labels := metricLabels(req, nil, 0)
	if len(labels) > 0 {
		t.Errorf("nil request should return empty labels")
		return
	}

	testAddr := "https://www.example.com"
	req, err := http.NewRequest(http.MethodPost, testAddr, nil)
	if err != nil {
		t.Errorf("New request should success, %v", err)
		return
	}
	tup, err := url.Parse(testAddr)
	if err != nil {
		t.Errorf("Parse test address, %v", err)
		return
	}
	labels = metricLabels(req, nil, 0)
	if len(labels) < 5 {
		t.Errorf("metric labels should contain 5 elements")
		return
	}
	if labels[0] != http.MethodPost || labels[1] != tup.Host || labels[2] != tup.Path {
		t.Errorf("incorrect metric labels, %v", labels)
		return
	}
	res := &http.Response{
		StatusCode: http.StatusAccepted,
		Request:    req,
	}
	labels = metricLabels(req, res, time.Second)
	if len(labels) < 5 {
		t.Errorf("metric labels should contain 5 elements")
		return
	}
	if labels[0] != http.MethodPost ||
		labels[1] != tup.Host ||
		labels[2] != tup.Path ||
		labels[3] != strconv.Itoa(res.StatusCode) {
		t.Errorf("incorrect metric labels, %v", labels)
		return
	}
}
