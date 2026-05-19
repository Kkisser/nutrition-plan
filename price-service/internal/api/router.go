package api

import "net/http"

func Router(runtime *Runtime) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", runtime.Health)
	mux.HandleFunc("/estimate", runtime.Estimate)
	mux.HandleFunc("/reload", runtime.Reload)
	mux.HandleFunc("/debug/last-requests", runtime.LastRequests)
	return mux
}
