package middleware

import (
	"cloudshell/pkg/log"
	"net/http"
	"runtime"
	"time"
)

func AddIncomingRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		then := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				CreateRequestLog(r).Info("request errored out")
			}
		}()
		next.ServeHTTP(w, r)
		duration := time.Now().Sub(then)
		CreateRequestLog(r).Infof("request completed in %vms", float64(duration.Nanoseconds())/1000000)
	})
}

// createRequestLog returns a logger with relevant request fields
func CreateRequestLog(r *http.Request, additionalFields ...map[string]interface{}) log.Logger {
	fields := map[string]interface{}{}
	if len(additionalFields) > 0 {
		fields = additionalFields[0]
	}
	if r != nil {
		fields["host"] = r.Host
		fields["remote_addr"] = r.RemoteAddr
		fields["method"] = r.Method
		fields["protocol"] = r.Proto
		fields["path"] = r.URL.Path
		fields["request_url"] = r.URL.String()
		fields["user_agent"] = r.UserAgent()
		fields["cookies"] = r.Cookies()
	}
	return log.WithFields(fields)
}

func CreateMemoryLog() log.Logger {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return log.WithFields(map[string]interface{}{
		"alloc":       memStats.Alloc,
		"heap_alloc":  memStats.HeapAlloc,
		"total_alloc": memStats.TotalAlloc,
		"sys_alloc":   memStats.Sys,
		"gc_count":    memStats.NumGC,
	})
}
