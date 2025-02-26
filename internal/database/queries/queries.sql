-- Users
-- name: CreateUser :one
INSERT INTO users (email, password)
VALUES ($1, $2)
RETURNING id, email, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

--------------------------------------------------------

-- Teams
-- name: CreateTeam :one
INSERT INTO teams (name)
VALUES ($1)
RETURNING id, name, created_at, updated_at;

-- name: AddUserToTeam :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES ($1, $2, $3);

-- name: RemoveUserFromTeam :exec
DELETE FROM team_members
WHERE team_id = $1 AND user_id = $2;

-- name: GetTeamMembers :many
SELECT users.id, users.email, team_members.role
FROM team_members
JOIN users ON users.id = team_members.user_id
WHERE team_members.team_id = $1;

--------------------------------------------------------

-- Projects
-- name: CreateProject :one
INSERT INTO projects (name, owner_id)
VALUES ($1, $2)
RETURNING id, name, owner_id, created_at, updated_at;

-- name: GetUserProjects :many
SELECT id, name, owner_id, created_at, updated_at
FROM projects
WHERE owner_id = $1;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;

--------------------------------------------------------

-- Issues
-- name: CreateIssue :one
INSERT INTO issues (project_id, title, description, status)
VALUES ($1, $2, $3, $4)
RETURNING id, project_id, title, description, status, created_at, updated_at;

-- name: GetProjectIssues :many
SELECT id, title, description, status, created_at, updated_at
FROM issues
WHERE project_id = $1;

-- name: UpdateIssueStatus :exec
UPDATE issues
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteIssue :exec
DELETE FROM issues WHERE id = $1;

--------------------------------------------------------

-- Tasks
-- name: CreateTask :one
INSERT INTO tasks (project_id, assignee_id, title, description, status, priority, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, project_id, assignee_id, title, status, priority, due_date, created_at, updated_at;

-- name: GetUserTasks :many
SELECT id, project_id, title, status, priority, due_date, created_at, updated_at
FROM tasks
WHERE assignee_id = $1;

-- name: UpdateTaskStatus :exec
UPDATE tasks
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;
