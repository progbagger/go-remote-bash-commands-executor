package api

import (
	"bytes"
	"context"
	"database/sql"
	"db"
	"encoding/json"
	"executor"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type CancelHandler struct {
	cancelFuncs map[uint64]context.CancelFunc
	locker      sync.Locker

	conn *db.Connection
}

type ExecuteHandler struct {
	cancelHandler *CancelHandler

	conn *db.Connection
}

type GetCommandsHandler struct {
	conn *db.Connection
}

type GetFullCommandHandler struct {
	conn *db.Connection
}

func (handler *CancelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlValues := r.URL.Query()
	stringId := urlValues.Get("id")

	id, err := strconv.ParseUint(stringId, 10, 64)
	if err != nil {
		writeBadRequestError(err, w, r)
		return
	}

	if err := handler.callAndDelete(id); err != nil {
		http.NotFound(w, r)
		return
	}

	w.Write([]byte("Stopped"))
}

func (handler *ExecuteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestBody := new(RequestBody)
	err := json.NewDecoder(r.Body).Decode(requestBody)
	if err != nil {
		writeBadRequestError(err, w, r)
		return
	}

	// writing database record
	id, err := handler.conn.InsertRecord(
		db.CommandTableRecord{Command: requestBody.Command},
		db.InputTableRecord{Input: requestBody.Input, Env: requestBody.Env},
	)
	if err != nil {
		log.Println(err)
		writeInternalServerError(err, w, r)
		return
	}

	// populating cancel functions map
	ctx, cancel := context.WithCancel(context.Background())
	handler.cancelHandler.insert(id, cancel)

	// preparing streams
	outWriter := new(bytes.Buffer)
	errWriter := new(bytes.Buffer)

	// launching command
	executor := executor.Executor{Workdir: requestBody.Workdir, Env: requestBody.Env}
	isDone := executor.RunScript(
		ctx,
		strings.NewReader(requestBody.Input),
		outWriter,
		errWriter,
		requestBody.Command,
	)
	log.Printf("launched command with id = %d", id)

	outputs := new(db.OutputsTableRecord)

	// launching gorouitne to watch for outputs changes
	go func(id uint64, isDone <-chan error) {
		for {
			select {
			case <-time.After(time.Second * 5):
				outputs.Output = outWriter.String()
				outputs.Errors = errWriter.String()
				handler.conn.UpdateRecord(
					id,
					outputs,
					db.StatusesTableRecord{ExitCode: -2},
				)

				log.Printf("command with id = %d is updated its outputs\n", id)
			case err := <-isDone:
				var statuses db.StatusesTableRecord
				if exitErr, ok := err.(*exec.ExitError); ok || err == nil {
					if ok {
						statuses.ExitCode = exitErr.ExitCode()
					} else {
						statuses.ExitCode = 0
					}

					if statuses.ExitCode == -1 {
						log.Printf("command with id = %d is interrupted\n", id)
					} else {
						log.Printf("command with id = %d is finished\n", id)
					}
				} else {
					statuses.ExitCode = -1
					log.Printf("command with id = %d isn't started\n", id)
				}

				outputs.Output = outWriter.String()
				outputs.Errors = errWriter.String()
				handler.conn.UpdateRecord(
					id,
					outputs,
					statuses,
				)

				// removing cancel function of this command
				handler.cancelHandler.callAndDelete(id)

				// returning to end goroutine
				return
			}
		}
	}(id, isDone)

	w.Write([]byte("Launched"))
}

func (handler *GetCommandsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	commands, err := handler.conn.GetCommands()
	if err != nil {
		writeInternalServerError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(&commands)
}

func (handler *GetFullCommandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlValues := r.URL.Query()
	stringId := urlValues.Get("id")

	id, err := strconv.ParseUint(stringId, 10, 64)
	if err != nil {
		writeBadRequestError(err, w, r)
		return
	}

	fullCommand, err := handler.conn.GetFullRecordById(id)
	if err == sql.ErrNoRows {
		writeBadRequestError(err, w, r)
		return
	}
	if err != nil {
		writeInternalServerError(err, w, r)
		return
	}

	json.NewEncoder(w).Encode(&fullCommand)
}

func NewCancelHandler(conn *db.Connection) (*CancelHandler, error) {
	if err := checkConnection(conn); err != nil {
		return nil, err
	}

	h := new(CancelHandler)
	h.cancelFuncs = make(map[uint64]context.CancelFunc)
	h.conn = conn
	h.locker = &sync.Mutex{}
	return h, nil
}

func NewExecuteHandler(conn *db.Connection, cancelHandler *CancelHandler) (*ExecuteHandler, error) {
	if err := checkConnection(conn); err != nil {
		return nil, err
	}
	if err := checkCancelHandler(cancelHandler); err != nil {
		return nil, err
	}

	h := new(ExecuteHandler)
	h.cancelHandler = cancelHandler
	h.conn = conn
	return h, nil
}

func NewGetCommandsHandler(conn *db.Connection) (*GetCommandsHandler, error) {
	if err := checkConnection(conn); err != nil {
		return nil, err
	}

	h := new(GetCommandsHandler)
	h.conn = conn
	return h, nil
}

func NewGetFullCommandHandler(conn *db.Connection) (*GetFullCommandHandler, error) {
	if err := checkConnection(conn); err != nil {
		return nil, err
	}

	h := new(GetFullCommandHandler)
	h.conn = conn
	return h, nil
}

type RequestBody struct {
	Workdir string                      `json:"workdir"`
	Env     []executor.EnvironmentEntry `json:"env"`

	Input   string `json:"input"`
	Command string `json:"command"`
}

func (cancelHandler *CancelHandler) insert(id uint64, cancelFunc context.CancelFunc) {
	cancelHandler.locker.Lock()
	defer cancelHandler.locker.Unlock()

	cancelHandler.cancelFuncs[id] = cancelFunc
}

func (cancelHandler *CancelHandler) delete(id uint64) {
	cancelHandler.locker.Lock()
	defer cancelHandler.locker.Unlock()

	delete(cancelHandler.cancelFuncs, id)
}

func (cancelHandler *CancelHandler) callAndDelete(id uint64) error {
	cancelHandler.locker.Lock()
	defer cancelHandler.locker.Unlock()

	if v, exists := cancelHandler.cancelFuncs[id]; exists {
		v()
		delete(cancelHandler.cancelFuncs, id)
	} else {
		return fmt.Errorf("no such id")
	}

	return nil
}

func checkConnection(conn *db.Connection) error {
	if conn == nil {
		return fmt.Errorf("connection can't be nil")
	}

	return nil
}

func checkCancelHandler(cancelHandler *CancelHandler) error {
	if cancelHandler == nil {
		return fmt.Errorf("cancel handler can't be nil")
	}

	return nil
}

func writeInternalServerError(err error, w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(fmt.Sprintf("500 Internal Server Error: %s", err.Error())))
}

func writeBadRequestError(err error, w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(fmt.Sprintf("400 Bad Request: %s", err.Error())))
}
