package utils

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
)

// 格式化字节
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatUint(bytes, 10) + " B"
	}
	exp := int(math.Log(float64(bytes)) / math.Log(float64(unit)))
	div := uint64(math.Pow(unit, float64(exp)))
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// 请求头跨域
func Cors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Cache-Control", "max-age=21600") // 设置缓存6小时
		return
	}
}
