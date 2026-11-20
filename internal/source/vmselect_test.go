package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVMSelect_Fetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"alertname":"Test","severity":"critical","cluster":"c1","namespace":"ns"},"value":[1700000000,"1"]}]}}`))
	}))
	defer ts.Close()
	v := &VMSelect{SelectURL: ts.URL}
	alerts, err := v.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 || alerts[0].Cluster != "c1" {
		t.Fatalf("%+v", alerts)
	}
}