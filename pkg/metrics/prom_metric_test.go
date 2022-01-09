package metrics

import (
	"testing"
)

func Test(t *testing.T) {

	//response := &struct {
	//	Status string `json:"status"`
	//	Data   struct {
	//		Result []struct {
	//			Metrics struct {
	//				Stage string `json:"instance"`
	//			} `json:"metric"`
	//			Values []interface{} `json:"value"`
	//		} `json:"result"`
	//	} `json:"data"`
	//}{}
	cache := NewPromCache()

	//response := ""
	err := cache.SyncTiKVWritingByte("default-basic", "127.0.0.1:9090")
	if err != nil {
		t.Error(err)
	}
}
