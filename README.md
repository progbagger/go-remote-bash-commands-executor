# go-remote-bash-commands-executor

Server that provides an **API** to execute bash-scripts remotely on the server and store its outputs in the database.

## Required OS - **Linux**

## Usage

API has a few endpoints:

- `/api/commands` - **GET** - fetches all launched commands
- `/api/get_command?id=<id>` - **GET** - gets full indormation about command with provided ID

Result will be looking like that (empty values will be omitted):

```json
[
  {
    "command_info": {
      "command": "command1"
    },
    "input_info": {
      "input": "input",
    },
    "outputs": {
      "output": "output",
      "errors": "errors"
    },
    "statuses": {
      "exit_code": exit_code
    }
  },
    {
    "command_info": {
      "command": "command2"
    },
    "input_info": {
      "input": "input",
    },
    "outputs": {
      "output": "output",
      "errors": "errors"
    },
    "statuses": {
      "exit_code": exit_code
    }
  }
]
```

- `/api/launch` - **POST** - launches new command on the server. Accepts request body:

```json
{
  "command": "echo $this_is_the_key | cat -",
  "env": [
    "key": "this_is_the_key",
    "value": "this_is_the_value"
  ],
  "input": "this_is_the_stdin",
  "workdir": "/home"
}
```

Only necessary parameter is `command`.

- `/api/cancel?id=<id>` - **POST** - cancels execution of the command with provided ID

If command is long enough, then **every 5 seconds** its *stdout* and *stderr* updates and sends into the database.

If command was cancelled or there are some errors on the server - exit code of this command will be **-1**.

## Database description

Database consists of 4 tables:

### `commands`

| field | type | key |
| ----- | ---- | --- |
| id | `SERIAL` | Primary Key |
| command | `TEXT NOT NULL` | |

### `inputs`

| field | type | key |
| ----- | ---- | --- |
| id | `SERIAL` | References `commands` (`id`) |
| input | `TEXT` | |
| env | `env_entry` | |

#### Type `env_entry`

| field | type |
| ----- | ---- |
| key | `TEXT` |
| value | `TEXT` |

### `outputs`

| field | type | key |
| ----- | ---- | --- |
| id | `SERIAL` | References `commands` (`id`) |
| output | `TEXT` | |
| errors | `TEXT` | |

### `statuses`

| field | type | key |
| ----- | ---- | --- |
| id | `SERIAL` | References `commands` (`id`) |
| exit_code | `INTEGER` | |

Upon succesful insertion into `commands` table appropriate amount of empty records are inserted into tables `outputs` and `statuses`.

## Launching

### docker-compose

Port for the database is **5432** and for the server is **8888**.

```shell
docker-compose up
```

## Questions and desicions

- What is a script will be?

> When we launching scripts or commands in the terminal we have a few stable envrionments:
>
> 1. working directory
> 2. environment variables
> 3. command or script itself
> 4. command's or script's stdin
> 5. command's or script's stdout and stderr
>
> So I decided to represent all of these things.
>
> We also have an arguments that can be passed to the command or script but in terms of launching them through `bash -c` they are working a little odd.

- Which tables and how many of them should I be using?

> In case of different API endpoints (for example - getting a list of launched commands) i needed to separate some of properties of the command. Thus for getting a list of all commands we need only commands themselves - the first and main table `commands` is defined.
>
> `inputs` table was born logically - it is separated from outputs and statuses table.
>
> Outputs are tethered because they can be merged or redirected in script - `outputs` table is born.
>
> And `statuses` table is separated from all of them by meaning that command is finished and server will no longer gather command's outputs.

- How to launch everything?

> In my opinion the simplest solution is **docker-compose**. I quite worked with it and it allows to start multiple services in one command, such as **builder**, **database** and **runner** itself.
