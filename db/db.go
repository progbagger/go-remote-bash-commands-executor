package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

const (
	DefaultUsername string = "postgres"
	DefaultPassword string = "postgres"
	DefaultDatabase string = "postgres"

	DefaultHost string = "localhost"
	DefaultPort int    = 5432
)

const DefaultTimeout = time.Second * 30

// Connection credentials.
//
// If anything is empty or 0 - it will be replaced with
// default constant values.
//
//	Default username - "postgres"
//	Default password - "postgres"
//	Default database - "postgres"
//	Default host - "localhost"
//	Default port - 5432
type Credentials struct {
	Username string
	Password string
	Database string

	Host string
	Port int
}

type Connection struct {
	db *sql.DB
}

func Open(credentials Credentials) (*Connection, error) {
	checkDefaultCredentials(&credentials)
	conn, err := sql.Open("postgres", fmt.Sprintf(
		"user='%s' password='%s' dbname='%s' host='%s' port='%d' connect_timeout=30",
		credentials.Username,
		credentials.Password,
		credentials.Database,
		credentials.Host,
		credentials.Port,
	))

	if err != nil {
		return nil, err
	}

	connection := new(Connection)
	connection.db = conn
	return connection, nil
}

func (connection *Connection) Close() error {
	return connection.db.Close()
}

func checkDefaultCredentials(credentials *Credentials) {
	if credentials.Username == "" {
		credentials.Username = DefaultUsername
	}
	if credentials.Password == "" {
		credentials.Password = DefaultPassword
	}
	if credentials.Host == "" {
		credentials.Host = DefaultHost
	}
	if credentials.Port == 0 {
		credentials.Port = DefaultPort
	}
}

const (
	RunningStatus     string = "RUNNING"
	FinishedStatus    string = "FINISHED"
	InterruptedStatus string = "INTERRUPTED"
)

// Struct that represents database command record
type CommandTableRecord struct {
	Id      int64    `json:"id"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type InputTableRecord struct {
	Id    int64  `json:"id"`
	Input string `json:"input"`
}

type OutputsTableRecord struct {
	Id int64 `json:"id"`

	Output string `json:"output"`
	Errors string `json:"errors"`
}

type StatusesTableRecord struct {
	Id int64 `json:"id"`

	ExitCode int    `json:"exit_code"`
	Status   string `json:"status"`
}

type FullCommandRecord struct {
	Command  CommandTableRecord  `json:"command_info"`
	Input    InputTableRecord    `json:"input_info"`
	Outputs  OutputsTableRecord  `json:"outputs"`
	Statuses StatusesTableRecord `json:"statuses"`
}

func (connection *Connection) GetCommands() ([]CommandTableRecord, error) {
	rows, err := connection.db.QueryContext(
		createTimeoutDefaultContext(),
		`SELECT * FROM commands`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []CommandTableRecord
	for rows.Next() {
		var record CommandTableRecord
		err := rows.Scan(
			&record.Id,
			&record.Command,
		)
		if err != nil {
			return records, err
		}

		records = append(records, record)
	}

	return records, nil
}

func (connection *Connection) GetRecordById(recordId int64) (FullCommandRecord, error) {
	var record FullCommandRecord

	row := connection.db.QueryRowContext(
		createTimeoutDefaultContext(),
		`
			SELECT id, c.command, i.input, o.output, o.errors, s.exit_code, s.status
			FROM commands AS c WHERE id = $1
			JOIN inputs AS i ON c.id = i.id
			JOIN outputs AS o ON c.id = o.id
			JOIN statuses AS s ON c.id = s.id
		`,
		recordId,
	)

	err := row.Scan(
		&record.Command.Id,
		&record.Command.Command,
		&record.Input.Input,
		&record.Outputs.Output,
		&record.Outputs.Errors,
		&record.Statuses.ExitCode,
		&record.Statuses.Status,
	)
	record.Input.Id = record.Command.Id
	record.Outputs.Id = record.Command.Id
	record.Statuses.Id = record.Command.Id

	return record, err
}

func (connection *Connection) InsertRecord(command CommandTableRecord, input InputTableRecord) (int64, error) {
	tx, err := connection.db.BeginTx(createTimeoutDefaultContext(), nil)
	if err != nil {
		return -1, err
	}

	row := tx.QueryRowContext(
		createTimeoutDefaultContext(),
		`INSERT INTO commands (command) VALUES ($1) RETURNING id`,
		command.Command,
	)
	err = row.Scan(&command.Id)
	if err != nil {
		tx.Rollback()
		return command.Id, err
	}

	_, err = tx.ExecContext(
		createTimeoutDefaultContext(),
		`INSERT INTO inputs VALUES ($1, $2)`,
		command.Id,
		input.Input,
	)
	if err != nil {
		tx.Rollback()
		return command.Id, err
	}

	return command.Id, tx.Commit()
}

func (connection *Connection) UpdateRecord(outputs *OutputsTableRecord, statuses StatusesTableRecord) error {
	// TODO
}

func createTimeoutDefaultContext() context.Context {
	ctx, _ := context.WithTimeoutCause(
		context.Background(),
		DefaultTimeout,
		fmt.Errorf("operation timed out"),
	)

	return ctx
}
