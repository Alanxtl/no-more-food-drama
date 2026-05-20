package amap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchAroundMapsAmapPois(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v5/place/around" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "amap-key" {
			t.Fatalf("missing key query")
		}
		if r.URL.Query().Get("location") != "113.320000,23.090000" {
			t.Fatalf("location = %q", r.URL.Query().Get("location"))
		}
		if r.URL.Query().Get("radius") != "3000" {
			t.Fatalf("radius = %q", r.URL.Query().Get("radius"))
		}
		if r.URL.Query().Get("types") != "050000" {
			t.Fatalf("types = %q", r.URL.Query().Get("types"))
		}
		if r.URL.Query().Get("page_size") != "20" {
			t.Fatalf("page_size = %q", r.URL.Query().Get("page_size"))
		}
		if r.URL.Query().Get("show_fields") != "business" {
			t.Fatalf("show_fields = %q", r.URL.Query().Get("show_fields"))
		}
		_, _ = w.Write([]byte(`{
			"status":"1",
			"pois":[
				{
					"id":"B0FFTEST",
					"name":"鮨小野",
					"address":"海珠区测试路 1 号",
					"location":"113.320000,23.090000",
					"distance":"650",
					"type":"餐饮服务;外国餐厅;日本料理",
					"biz_ext":{"rating":"4.7","cost":"128"}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewClient("amap-key", server.URL+"/", server.Client())
	restaurants, err := client.SearchAround(context.Background(), SearchRequest{Lat: 23.09, Lng: 113.32, RadiusMeters: 3000, Limit: 20})
	if err != nil {
		t.Fatalf("SearchAround returned error: %v", err)
	}

	if len(restaurants) != 1 {
		t.Fatalf("restaurants length = %d", len(restaurants))
	}
	got := restaurants[0]
	if got.ID != "amap:B0FFTEST" || got.Provider != "amap" || got.ProviderID != "B0FFTEST" {
		t.Fatalf("provider identity = %#v", got)
	}
	if got.Name != "鮨小野" || got.Address != "海珠区测试路 1 号" {
		t.Fatalf("restaurant text = %#v", got)
	}
	if got.Lat != 23.09 || got.Lng != 113.32 || got.DistanceMeters != 650 {
		t.Fatalf("restaurant location = %#v", got)
	}
	if got.AvgPriceCNY != 128 || got.Rating != 4.7 {
		t.Fatalf("restaurant facts = %#v", got)
	}
	if len(got.Categories) != 3 || got.Categories[2] != "日本料理" {
		t.Fatalf("categories = %#v", got.Categories)
	}
	if len(got.TypeIDs) != 0 || len(got.Tags) != 0 {
		t.Fatalf("classification fields = %#v %#v", got.TypeIDs, got.Tags)
	}
}

func TestSearchAroundReturnsAmapStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"0","info":"INVALID_USER_KEY","pois":[]}`))
	}))
	defer server.Close()

	client := NewClient("bad-key", server.URL, server.Client())
	if _, err := client.SearchAround(context.Background(), SearchRequest{}); err == nil {
		t.Fatal("SearchAround returned nil error")
	}
}
