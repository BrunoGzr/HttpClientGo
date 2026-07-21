package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiHandler struct{}

type Server struct {
	addr    string
	Handler http.Handler
}

func (apiHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}

func healthzHandler(Writer http.ResponseWriter, request *http.Request) {
	Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	Writer.WriteHeader(200)
	Writer.Write([]byte("OK"))
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
func (cfg *apiConfig) metricsHandler(Writer http.ResponseWriter, request *http.Request) {
	Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	Writer.WriteHeader(200)
	Writer.Write([]byte(fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", cfg.fileserverHits.Load())))
}

func (cfg *apiConfig) resetmetricsHandler(Writer http.ResponseWriter, request *http.Request) {
	Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	Writer.WriteHeader(200)
	cfg.fileserverHits.CompareAndSwap(cfg.fileserverHits.Load(), 0)
	Writer.Write([]byte("Metrics reseted."))
}

func (cfg *apiConfig) jsonHandler(Writer http.ResponseWriter, request *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	params := parameters{}
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&params)
	if err != nil {
		return
	}

	if len(params.Body) > 140 {
		type errorResponse struct {
			Error string `json:"error"`
		}
		respBody := errorResponse{
			Error: "Chirp is too long",
		}
		data, err2 := json.Marshal(respBody)
		if err2 != nil {
			return
		}
		Writer.Header().Set("Content-Type", "application/json")
		Writer.WriteHeader(400)
		Writer.Write(data)
	} else {
		msg := validateMessage(params.Body)
		type validParams struct {
			Cleaned_body string `json:"cleaned_body"`
		}
		response := validParams{
			Cleaned_body: msg,
		}
		data, err2 := json.Marshal(response)
		if err2 != nil {
			return
		}
		Writer.Header().Set("Content-Type", "application/json")
		Writer.WriteHeader(200)
		Writer.Write(data)
	}

}

func validateMessage(Unfilteredmsg string) string {
	msg := Unfilteredmsg
	newMsg := []string{}
	for _, word := range strings.Split(msg, " ") {
		word2 := strings.ToLower(word)
		if word2 == "kerfuffle" || word2 == "sharbert" || word2 == "fornax" {
			newMsg = append(newMsg, "****")
			continue
		}
		newMsg = append(newMsg, word)
	}
	return strings.Join(newMsg, " ")
}

func main() {
	metrics := apiConfig{}
	const port = ":8080"
	mux := http.NewServeMux()
	mux.Handle("/app/", metrics.middlewareMetricsInc((http.StripPrefix("/app/", http.FileServer(http.Dir("."))))))
	srv := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	mux.HandleFunc("POST /api/validate_chirp", metrics.jsonHandler)
	mux.HandleFunc("POST /admin/reset", metrics.resetmetricsHandler)
	mux.HandleFunc("GET /admin/metrics", metrics.metricsHandler)

	mux.HandleFunc("GET /api/healthz", healthzHandler)
	log.Println("Listening on " + port)
	log.Fatal(srv.ListenAndServe())

}
