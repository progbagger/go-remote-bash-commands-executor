package db

import (
	"common"
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Default postgresql connection values
const (
	DefaultUsername string = "postgres"
	DefaultPassword string = "postgres"
	DefaultDatabase string = "postgres"

	DefaultHost string = "localhost"
	DefaultPort int    = 5432
)

const defaultTimeout = time.Second * 30

// Connection credentials.
//
// If anything is empty or 0 - it will be replaced with
// default values.
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

// Struct that represents specified postgresql connection with specified
// methods. Closing it isn't necessary.
type Connection struct {
	db *sql.DB
}

// Creates new Connection with provided credentials.
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

// Closes opened connection. Call to this method is not necessary.
func (connection *Connection) Close() error {
	return connection.db.Close()
}

// Returns an array of command strings stored in the database.
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

// Returns fully populated data of launched or finished command that stores
// in the database.
func (connection *Connection) GetFullRecordById(recordId int64) (FullCommandRecord, error) {
	var record FullCommandRecord

	row := connection.db.QueryRowContext(
		createTimeoutDefaultContext(),
		`
			SELECT c.command, i.input, i.args, i.env, o.output, o.errors, s.exit_code, s.status
			FROM commands AS c WHERE id = $1
			JOIN inputs AS i ON c.id = i.id
			JOIN outputs AS o ON c.id = o.id
			JOIN statuses AS s ON c.id = s.id
		`,
		recordId,
	)

	err := row.Scan(
		&record.Command.Command,
		&record.Input.Input,
		&record.Input.Args,
		&record.Input.Env,
		&record.Outputs.Output,
		&record.Outputs.Errors,
		&record.Statuses.ExitCode,
		&record.Statuses.Status,
	)
	record.Command.Id = recordId
	record.Input.id = record.Command.Id
	record.Outputs.id = record.Command.Id
	record.Statuses.id = record.Command.Id

	return record, err
}

// Pushes command and its inputs into the database.
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
		`INSERT INTO inputs VALUES ($1, $2, $3, $4)`,
		command.Id,
		input.Input,
		input.Args,
		input.Env,
	)
	if err != nil {
		tx.Rollback()
		return command.Id, err
	}

	return command.Id, tx.Commit()
}

// Updates launched command's outputs and statuses.
func (connection *Connection) UpdateRecord(outputs *OutputsTableRecord, statuses StatusesTableRecord) error {
	tx, err := connection.db.BeginTx(createTimeoutDefaultContext(), nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		createTimeoutDefaultContext(),
		`
			UPDATE outputs SET output = $2, errors = $3
			WHERE id = $1
		`,
		outputs.id,
		outputs.Output,
		outputs.Errors,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(
		createTimeoutDefaultContext(),
		`
			UPDATE statuses SET exit_code = $2, status = $3
			WHERE id = $1
		`,
		statuses.id,
		statuses.ExitCode,
		statuses.Status,
	)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

const (
	RunningStatus     string = "RUNNING"
	FinishedStatus    string = "FINISHED"
	InterruptedStatus string = "INTERRUPTED"
)

// Struct that represents essential command info in the "commands" table.
type CommandTableRecord struct {
	Id int64 `json:"id"`

	Command string `json:"command"`
}

// Struct that represents command's inputs in the "inputs" table.
type InputTableRecord struct {
	id int64

	Input string                    `json:"input"`
	Args  []string                  `json:"args"`
	Env   []common.EnvironmentEntry `json:"env"`
}

// Struct that represents command's outputs in the "outputs" table.
type OutputsTableRecord struct {
	id int64

	Output string `json:"output"`
	Errors string `json:"errors"`
}

// Struct that represents command's results in the "statuses" table.
type StatusesTableRecord struct {
	id int64

	ExitCode int    `json:"exit_code"`
	Status   string `json:"status"`
}

// Struct that stores full command info.
type FullCommandRecord struct {
	Command  CommandTableRecord  `json:"command_info"`
	Input    InputTableRecord    `json:"input_info"`
	Outputs  OutputsTableRecord  `json:"outputs"`
	Statuses StatusesTableRecord `json:"statuses"`
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

func createTimeoutDefaultContext() context.Context {
	ctx, _ := context.WithTimeoutCause(
		context.Background(),
		defaultTimeout,
		fmt.Errorf("operation timed out"),
	)

	return ctx
}
