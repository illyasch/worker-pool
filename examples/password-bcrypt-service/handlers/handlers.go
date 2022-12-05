// Package handlers manages the different versions of the API.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/illyasch/worker-pool/pool"
)

var (
	ErrScheduleTimeout = fmt.Errorf("task scheduling timeout")
)

// APIConfig contains all the mandatory systems required by handlers.
type APIConfig struct {
	BusyTimeout    time.Duration
	Log            *log.Logger
	Workers        *pool.NonBlocking[string]
	PasswordMinLen int
}

type response struct {
	Error string `json:"error,omitempty"`
	Hash  string `json:"hash"`
}

type bcryptTask struct {
	log      *log.Logger
	password string
}

// Router constructs a http.Handler with all application routes defined.
func (cfg APIConfig) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/bcrypt", cfg.handleBcrypt)

	return mux
}

func (cfg APIConfig) handleBcrypt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		cfg.respond(w, http.StatusMethodNotAllowed, response{Error: http.StatusText(http.StatusMethodNotAllowed)})
		cfg.Log.Println("bcrypt", "ERROR", fmt.Errorf("incorrect request method %s", r.Method))
		return
	}

	pwd := r.FormValue("password")
	if len(pwd) < cfg.PasswordMinLen {
		err := errors.New("input password is incorrect")

		cfg.respond(w, http.StatusBadRequest, response{Error: err.Error()})
		cfg.Log.Println("bcrypt", "ERROR", fmt.Errorf("validation password(%s): %w", pwd, err))
		return
	}

	hash, err := cfg.scheduleBcrypt(pwd)
	if err == nil {
		cfg.respond(w, http.StatusOK, response{Hash: hash})
		cfg.Log.Println("bcrypt", "statusCode", http.StatusOK, "method", r.Method, "path", r.URL.Path, "remoteaddr", r.RemoteAddr)
		return
	}

	if errors.Is(err, ErrScheduleTimeout) {
		cfg.respond(w, http.StatusTooManyRequests, response{Error: http.StatusText(http.StatusTooManyRequests)})
	} else {
		cfg.respond(w, http.StatusInternalServerError, response{Error: http.StatusText(http.StatusInternalServerError)})
	}

	cfg.Log.Println("bcrypt", "ERROR", fmt.Errorf("bcrypt: %w", err))
	return
}

// scheduleBcrypt sends a request for execution of a bcrypt task to a free worker.
// If there is no available worker or a task execution takes longer than cfg.BusyTimeout,
// it returns ErrScheduleTimeout.
func (cfg APIConfig) scheduleBcrypt(pwd string) (string, error) {
	task := bcryptTask{
		log:      cfg.Log,
		password: pwd,
	}

	// Gets a channel to get a free worker.
	requests := cfg.Workers.RequestChan()
	timer := time.NewTimer(cfg.BusyTimeout)
	defer timer.Stop()

	select {
	// Retrieves a first free worker for a task execution.
	case req := <-requests:
		select {
		// Sends a task to the worker then worker starts the task execution.
		case req.Request <- task:
			select {
			// Waits until the task execution is finished and retrieves a response struct.
			case resp := <-req.Response:
				return resp.Value, resp.Err
			case <-timer.C:
			}

		case <-timer.C:
		}

	case <-timer.C:
	}

	return "", ErrScheduleTimeout
}

func (r bcryptTask) Job(context.Context) pool.JobResponse[string] {
	hash, err := bcrypt.GenerateFromPassword([]byte(r.password), bcrypt.DefaultCost)
	if err != nil {
		r.log.Println("bcrypt password", "ERROR", err)
		return pool.JobResponse[string]{Err: fmt.Errorf("bcript.GenerateFromPassword: %w", err)}
	}

	r.log.Println("bcrypt password", "SUCCESS")
	return pool.JobResponse[string]{Value: string(hash)}
}

func (cfg APIConfig) respond(w http.ResponseWriter, statusCode int, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		cfg.Log.Println("respond", "ERROR", fmt.Errorf("json marshal: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(jsonData); err != nil {
		cfg.Log.Println("respond", "ERROR", fmt.Errorf("write output: %w", err))
		return
	}
}
