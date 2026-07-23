package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Brunogzr/HttpServerGo/internal/auth"
	"github.com/Brunogzr/HttpServerGo/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
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
	*database.Queries
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

func (cfg *apiConfig) chirpsPostHandler(Writer http.ResponseWriter, request *http.Request) {
	type parameters struct {
		Body    string    `json:"body"`
		User_id uuid.UUID `json:"user_id"`
	}
	params := parameters{}
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
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

		parametersToInsert := database.InsertChirpsParams{
			Body:   sql.NullString{String: params.Body, Valid: true},
			UserID: uuid.NullUUID{UUID: params.User_id, Valid: true},
		}
		msg, err := cfg.Queries.InsertChirps(request.Context(), parametersToInsert)
		if err != nil {
			fmt.Println(err)
			return
		}
		Writer.Header().Set("Content-Type", "application/json")
		Writer.WriteHeader(201)
		type msgValidated struct {
			Id        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}
		msgValid := msgValidated{
			Id:        msg.ID,
			CreatedAt: msg.CreatedAt.Time,
			UpdatedAt: msg.UpdatedAt.Time,
			Body:      msg.Body.String,
			UserID:    msg.UserID.UUID,
		}
		err = json.NewEncoder(Writer).Encode(msgValid)
		if err != nil {
			fmt.Println(err)
			return
		}

	}

}

func (cfg *apiConfig) usersLoginHandler(Writer http.ResponseWriter, request *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	params := parameters{}
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		return
	}
	email := sql.NullString{params.Email, true}
	user, err := cfg.SearchByEmail(request.Context(), email)
	if err != nil {
		fmt.Println(err)
		return
	}
	checkedPassword, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if checkedPassword {
		Writer.WriteHeader(200)
		type userValidated struct {
			Id        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
		}

		userResponse := userValidated{
			Id:        user.ID,
			CreatedAt: user.CreatedAt.Time,
			UpdatedAt: user.UpdatedAt.Time,
			Email:     user.Email.String,
		}
		Writer.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(Writer).Encode(userResponse)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		Writer.WriteHeader(401)
		Writer.Header().Set("Content-Type", "plain/text; charset=utf-8")
		Writer.Write([]byte("Incorrect email or password"))
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

func (cfg *apiConfig) usersCreateHandler(Writer http.ResponseWriter, request *http.Request) {
	type users struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Password  string    `json:"password"`
	}

	params := users{}
	decoder := json.NewDecoder(request.Body)
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Println(err)
		return
	}
	passwordHashed, err := auth.HashPassword(params.Password)
	if err != nil {
		fmt.Println(err)
		return
	}

	userParams := database.CreateUserParams{
		Email:          sql.NullString{params.Email, true},
		HashedPassword: passwordHashed,
	}

	userCreated, err := cfg.Queries.CreateUser(request.Context(), userParams)
	if err != nil {
		log.Fatal(err)
	}
	Writer.Header().Set("Content-Type", "application/json")
	Writer.WriteHeader(201)
	user := users{
		ID:        userCreated.ID,
		CreatedAt: userCreated.CreatedAt.Time,
		UpdatedAt: userCreated.UpdatedAt.Time,
		Email:     userCreated.Email.String,
	}
	err = json.NewEncoder(Writer).Encode(user)
	if err != nil {
		log.Fatal(err)
		return
	}
}

func (cfg *apiConfig) usersResetHandler(Writer http.ResponseWriter, request *http.Request) {
	if os.Getenv("PLATAFORM") == "DEV" {
		cfg.Queries.DeleteAllProducts(request.Context())
		Writer.WriteHeader(200)
	} else {
		Writer.WriteHeader(401)
		Writer.Write([]byte("401 Unauthorized\n"))
	}
}

func (cfg *apiConfig) chirpsFindAllHandler(Writer http.ResponseWriter, request *http.Request) {
	type chirpsStruct struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		User_id   uuid.UUID `json:"user_id"`
	}
	params := []chirpsStruct{}
	chirps, err := cfg.FindallChirps(request.Context())
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, chirp := range chirps {
		params = append(params, chirpsStruct{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt.Time,
			UpdatedAt: chirp.UpdatedAt.Time,
			Body:      chirp.Body.String,
			User_id:   chirp.UserID.UUID,
		})
	}
	Writer.WriteHeader(200)
	err = json.NewEncoder(Writer).Encode(params)
	if err != nil {
		fmt.Println(err)
		return
	}

}

func (cfg *apiConfig) chirpsFindByIdHandler(Writer http.ResponseWriter, request *http.Request) {
	type chirpResponseType struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		User_id   uuid.UUID `json:"user_id"`
	}
	requestId := request.PathValue("chirpID")
	id, err := uuid.Parse(requestId)
	if err != nil {
		fmt.Println(err)
		return
	}

	chirp, err := cfg.ChirpsFindById(request.Context(), id)

	if errors.Is(err, sql.ErrNoRows) {
		Writer.WriteHeader(404)
		Writer.Write([]byte(nil))
	} else {
		chirpResponse := chirpResponseType{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt.Time,
			UpdatedAt: chirp.UpdatedAt.Time,
			Body:      chirp.Body.String,
			User_id:   chirp.UserID.UUID,
		}
		Writer.WriteHeader(200)
		err = json.NewEncoder(Writer).Encode(chirpResponse)
	}
}

func main() {

	errDotFiles := godotenv.Load()
	if errDotFiles != nil {
		log.Fatal("Error loading .env file")
		return
	}
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
		return
	}
	dbQueries := database.New(db)
	metrics := apiConfig{Queries: dbQueries}
	const port = ":8080"
	mux := http.NewServeMux()
	mux.Handle("/app/", metrics.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	srv := &http.Server{
		Addr:    port,
		Handler: mux,
	}

	mux.HandleFunc("GET /api/chirps/{chirpID}", metrics.chirpsFindByIdHandler)
	mux.HandleFunc("GET /api/chirps", metrics.chirpsFindAllHandler)
	mux.HandleFunc("POST /api/login", metrics.usersLoginHandler)
	mux.HandleFunc("POST /api/users", metrics.usersCreateHandler)
	mux.HandleFunc("POST /api/chirps", metrics.chirpsPostHandler)
	mux.HandleFunc("POST /admin/reset", metrics.usersResetHandler)
	mux.HandleFunc("GET /admin/metrics", metrics.metricsHandler)

	mux.HandleFunc("GET /api/healthz", healthzHandler)
	log.Println("Listening on " + port)
	log.Fatal(srv.ListenAndServe())

}
