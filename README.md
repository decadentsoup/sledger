# Sledger
 
## What is Sledger?

Sledger, short for "schema ledger", is a tool for managing database migrations. It is currently designed specifically for PostgreSQL but can be expanded to other databases in the future.

The general design principle is you maintain a ledger of changes to the database with optional rollback information. You can either append new transactions to the end of the ledger or delete transactions from the end of the ledger, and then you can run a "sync" operation to get the database to match the state of the ledger. The database maintains its own copy of the ledger used (1) for rollback events (if applicable) and (2) to ensure integrity of the ledger file.

It is designed to be used in a GitOps environment where entries are added with commits and removed with reversions or by moving tags. However, it is agnostic of version control system and can be used without a VCS entirely.

## Why is Sledger?

I have worked with similar tools before, but generally found that they did not have a focus on GitOps and, as such, a lot of boilerplate needed to be set up to get it working. Sledger was built with GitOps from the start and, as such, can be easily implemented in a GitOps infrastructure.

## How do I use Sledger?

Create a YAML or JSON file. It will contain a root property called `sledger` with an array of commands indicating the SQL queries you want to run. For example:

```yaml
sledger:
  - forward: CREATE TABLE account (username TEXT NOT NULL, password TEXT NOT NULL);
  - forward: CREATE TABLE post (title TEXT NOT NULL, body TEXT NOT NULL);
```

If you want to specify rollback commands, you can use the `backward` property as below:

```yaml
sledger:
  - forward: CREATE TABLE account (username TEXT NOT NULL, password TEXT NOT NULL);
    backward: DROP TABLE account;
  - forward: CREATE TABLE post (title TEXT NOT NULL, body TEXT NOT NULL);
    backward: DROP TABLE post;
```

Once you have your commands set, you can simply run `sledger --ledger path/to/yaml --database database-url` and it will synchronize your database to the ledger.

## What are the command-line options?

- `--ledger`: path within the git repository to the sledger file, defaults to `sledger.yaml`
- `--database`: URL of the database to update, defaults to `postgresql://localhost`

## How are changes applied to the database?

The first time Sledger is run, it will create a database table in the public schema titled `sledger`. This table contains a copy of every ledger entry it has implemented in order, including rollback information (if provided).

Sledger will first check integrity by going through the Sledger entries in the YAML and database and making sure they line up. If there are more entries in the YAML file, it will start executing those changes and updating the database's copy of the ledger. If there are more entries in the database, however, Sledger will start reverting changes in the reverse order they appear.

## Can I use Docker?

Yes! Here's an example:

```sh
docker run --rm -it --net=host -v "$PWD/myrepo":/ledger decadentsoup/sledger
```
