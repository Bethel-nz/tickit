// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: queries.sql

package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const addUserToTeam = `-- name: AddUserToTeam :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES ($1, $2, $3)
`

type AddUserToTeamParams struct {
	TeamID pgtype.UUID
	UserID pgtype.UUID
	Role   pgtype.Text
}

func (q *Queries) AddUserToTeam(ctx context.Context, arg AddUserToTeamParams) error {
	_, err := q.db.Exec(ctx, addUserToTeam, arg.TeamID, arg.UserID, arg.Role)
	return err
}

const createIssue = `-- name: CreateIssue :one

INSERT INTO issues (project_id, title, description, status)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, title, description, status, created_at, updated_at
`

type CreateIssueParams struct {
	ProjectID   pgtype.UUID
	Title       string
	Description pgtype.Text
	Status      pgtype.Text
}

// ------------------------------------------------------
// Issues
func (q *Queries) CreateIssue(ctx context.Context, arg CreateIssueParams) (Issue, error) {
	row := q.db.QueryRow(ctx, createIssue,
		arg.ProjectID,
		arg.Title,
		arg.Description,
		arg.Status,
	)
	var i Issue
	err := row.Scan(
		&i.ID,
		&i.ProjectID,
		&i.Title,
		&i.Description,
		&i.Status,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createProject = `-- name: CreateProject :one

INSERT INTO projects (name, owner_id)
VALUES ($1, $2)
RETURNING id, name, owner_id, created_at, updated_at
`

type CreateProjectParams struct {
	Name    string
	OwnerID pgtype.UUID
}

// ------------------------------------------------------
// Projects
func (q *Queries) CreateProject(ctx context.Context, arg CreateProjectParams) (Project, error) {
	row := q.db.QueryRow(ctx, createProject, arg.Name, arg.OwnerID)
	var i Project
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.OwnerID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createTask = `-- name: CreateTask :one

INSERT INTO tasks (project_id, assignee_id, title, description, status, priority, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, project_id, assignee_id, title, status, priority, due_date, created_at, updated_at
`

type CreateTaskParams struct {
	ProjectID   pgtype.UUID
	AssigneeID  pgtype.UUID
	Title       string
	Description pgtype.Text
	Status      pgtype.Text
	Priority    pgtype.Text
	DueDate     pgtype.Timestamp
}

type CreateTaskRow struct {
	ID         pgtype.UUID
	ProjectID  pgtype.UUID
	AssigneeID pgtype.UUID
	Title      string
	Status     pgtype.Text
	Priority   pgtype.Text
	DueDate    pgtype.Timestamp
	CreatedAt  pgtype.Timestamp
	UpdatedAt  pgtype.Timestamp
}

// ------------------------------------------------------
// Tasks
func (q *Queries) CreateTask(ctx context.Context, arg CreateTaskParams) (CreateTaskRow, error) {
	row := q.db.QueryRow(ctx, createTask,
		arg.ProjectID,
		arg.AssigneeID,
		arg.Title,
		arg.Description,
		arg.Status,
		arg.Priority,
		arg.DueDate,
	)
	var i CreateTaskRow
	err := row.Scan(
		&i.ID,
		&i.ProjectID,
		&i.AssigneeID,
		&i.Title,
		&i.Status,
		&i.Priority,
		&i.DueDate,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createTeam = `-- name: CreateTeam :one

INSERT INTO teams (name)
VALUES ($1)
RETURNING id, name, created_at, updated_at
`

// ------------------------------------------------------
// Teams
func (q *Queries) CreateTeam(ctx context.Context, name string) (Team, error) {
	row := q.db.QueryRow(ctx, createTeam, name)
	var i Team
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO users (email, password)
VALUES ($1, $2)
RETURNING id, email, created_at, updated_at
`

type CreateUserParams struct {
	Email    string
	Password string
}

type CreateUserRow struct {
	ID        pgtype.UUID
	Email     string
	CreatedAt pgtype.Timestamp
	UpdatedAt pgtype.Timestamp
}

// Users
func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	row := q.db.QueryRow(ctx, createUser, arg.Email, arg.Password)
	var i CreateUserRow
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const deleteIssue = `-- name: DeleteIssue :exec
DELETE FROM issues WHERE id = $1
`

func (q *Queries) DeleteIssue(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteIssue, id)
	return err
}

const deleteProject = `-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1
`

func (q *Queries) DeleteProject(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteProject, id)
	return err
}

const deleteTask = `-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1
`

func (q *Queries) DeleteTask(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteTask, id)
	return err
}

const deleteUser = `-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1
`

func (q *Queries) DeleteUser(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteUser, id)
	return err
}

const getProjectIssues = `-- name: GetProjectIssues :many
SELECT id, title, description, status, created_at, updated_at
FROM issues
WHERE project_id = $1
`

type GetProjectIssuesRow struct {
	ID          pgtype.UUID
	Title       string
	Description pgtype.Text
	Status      pgtype.Text
	CreatedAt   pgtype.Timestamp
	UpdatedAt   pgtype.Timestamp
}

func (q *Queries) GetProjectIssues(ctx context.Context, projectID pgtype.UUID) ([]GetProjectIssuesRow, error) {
	rows, err := q.db.Query(ctx, getProjectIssues, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetProjectIssuesRow
	for rows.Next() {
		var i GetProjectIssuesRow
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Description,
			&i.Status,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTeamMembers = `-- name: GetTeamMembers :many
SELECT users.id, users.email, team_members.role
FROM team_members
JOIN users ON users.id = team_members.user_id
WHERE team_members.team_id = $1
`

type GetTeamMembersRow struct {
	ID    pgtype.UUID
	Email string
	Role  pgtype.Text
}

func (q *Queries) GetTeamMembers(ctx context.Context, teamID pgtype.UUID) ([]GetTeamMembersRow, error) {
	rows, err := q.db.Query(ctx, getTeamMembers, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetTeamMembersRow
	for rows.Next() {
		var i GetTeamMembersRow
		if err := rows.Scan(&i.ID, &i.Email, &i.Role); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT id, email, password, created_at, updated_at
FROM users
WHERE email = $1
`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRow(ctx, getUserByEmail, email)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.Password,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByID = `-- name: GetUserByID :one
SELECT id, email, created_at, updated_at
FROM users
WHERE id = $1
`

type GetUserByIDRow struct {
	ID        pgtype.UUID
	Email     string
	CreatedAt pgtype.Timestamp
	UpdatedAt pgtype.Timestamp
}

func (q *Queries) GetUserByID(ctx context.Context, id pgtype.UUID) (GetUserByIDRow, error) {
	row := q.db.QueryRow(ctx, getUserByID, id)
	var i GetUserByIDRow
	err := row.Scan(
		&i.ID,
		&i.Email,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserProjects = `-- name: GetUserProjects :many
SELECT id, name, owner_id, created_at, updated_at
FROM projects
WHERE owner_id = $1
`

func (q *Queries) GetUserProjects(ctx context.Context, ownerID pgtype.UUID) ([]Project, error) {
	rows, err := q.db.Query(ctx, getUserProjects, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Project
	for rows.Next() {
		var i Project
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.OwnerID,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserTasks = `-- name: GetUserTasks :many
SELECT id, project_id, title, status, priority, due_date, created_at, updated_at
FROM tasks
WHERE assignee_id = $1
`

type GetUserTasksRow struct {
	ID        pgtype.UUID
	ProjectID pgtype.UUID
	Title     string
	Status    pgtype.Text
	Priority  pgtype.Text
	DueDate   pgtype.Timestamp
	CreatedAt pgtype.Timestamp
	UpdatedAt pgtype.Timestamp
}

func (q *Queries) GetUserTasks(ctx context.Context, assigneeID pgtype.UUID) ([]GetUserTasksRow, error) {
	rows, err := q.db.Query(ctx, getUserTasks, assigneeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetUserTasksRow
	for rows.Next() {
		var i GetUserTasksRow
		if err := rows.Scan(
			&i.ID,
			&i.ProjectID,
			&i.Title,
			&i.Status,
			&i.Priority,
			&i.DueDate,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const removeUserFromTeam = `-- name: RemoveUserFromTeam :exec
DELETE FROM team_members
WHERE team_id = $1 AND user_id = $2
`

type RemoveUserFromTeamParams struct {
	TeamID pgtype.UUID
	UserID pgtype.UUID
}

func (q *Queries) RemoveUserFromTeam(ctx context.Context, arg RemoveUserFromTeamParams) error {
	_, err := q.db.Exec(ctx, removeUserFromTeam, arg.TeamID, arg.UserID)
	return err
}

const updateIssueStatus = `-- name: UpdateIssueStatus :exec
UPDATE issues
SET status = $2, updated_at = now()
WHERE id = $1
`

type UpdateIssueStatusParams struct {
	ID     pgtype.UUID
	Status pgtype.Text
}

func (q *Queries) UpdateIssueStatus(ctx context.Context, arg UpdateIssueStatusParams) error {
	_, err := q.db.Exec(ctx, updateIssueStatus, arg.ID, arg.Status)
	return err
}

const updateTaskStatus = `-- name: UpdateTaskStatus :exec
UPDATE tasks
SET status = $2, updated_at = now()
WHERE id = $1
`

type UpdateTaskStatusParams struct {
	ID     pgtype.UUID
	Status pgtype.Text
}

func (q *Queries) UpdateTaskStatus(ctx context.Context, arg UpdateTaskStatusParams) error {
	_, err := q.db.Exec(ctx, updateTaskStatus, arg.ID, arg.Status)
	return err
}

const updateUserPassword = `-- name: UpdateUserPassword :exec
UPDATE users
SET password = $2, updated_at = now()
WHERE id = $1
`

type UpdateUserPasswordParams struct {
	ID       pgtype.UUID
	Password string
}

func (q *Queries) UpdateUserPassword(ctx context.Context, arg UpdateUserPasswordParams) error {
	_, err := q.db.Exec(ctx, updateUserPassword, arg.ID, arg.Password)
	return err
}
