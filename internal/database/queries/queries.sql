-- Users
-- name: CreateUser :one
INSERT INTO users (email, password, name, username, avatar_url, bio)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, email, name, username, avatar_url, bio, email_verified, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password, name, username, avatar_url, bio, email_verified, last_login_at, account_status, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, name, username, avatar_url, bio, email_verified, last_login_at, account_status, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, email, name, username, avatar_url, bio, email_verified, last_login_at, account_status, created_at, updated_at
FROM users
WHERE username = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetActiveProjectsCount :one
SELECT COUNT(*) 
FROM projects 
WHERE owner_id = $1 AND status = 'active';

-- name: UpdateUserProfile :exec
UPDATE users
SET 
  name = COALESCE($2, name),
  email = COALESCE($3, email),
  username = COALESCE($4, username),
  avatar_url = COALESCE($5, avatar_url),
  bio = COALESCE($6, bio),
  updated_at = now()
WHERE id = $1;

-- name: GetUserProfile :one
SELECT id, email, name, username, avatar_url, bio, email_verified, created_at, updated_at
FROM users
WHERE id = $1;

-- name: VerifyUserEmail :exec
UPDATE users
SET email_verified = true, updated_at = now()
WHERE id = $1;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = now(), updated_at = now()
WHERE id = $1;

-- name: UpdateUserAccountStatus :exec
UPDATE users
SET account_status = $2, updated_at = now()
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, username, avatar_url, email_verified, account_status, created_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

--------------------------------------------------------
-- Teams
-- name: CreateTeam :one
INSERT INTO teams (name, description, avatar_url)
VALUES ($1, $2, $3)
RETURNING id, name, description, avatar_url, created_at, updated_at;

-- name: GetTeamByID :one
SELECT id, name, description, avatar_url, created_at, updated_at
FROM teams
WHERE id = $1;

-- name: UpdateTeam :exec
UPDATE teams
SET 
  name = COALESCE($2, name),
  description = COALESCE($3, description),
  avatar_url = COALESCE($4, avatar_url),
  updated_at = now()
WHERE id = $1;

-- name: DeleteTeam :exec
DELETE FROM teams WHERE id = $1;

-- name: AddUserToTeam :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES ($1, $2, $3);

-- name: RemoveUserFromTeam :exec
DELETE FROM team_members
WHERE team_id = $1 AND user_id = $2;

-- name: UpdateTeamMemberRole :exec
UPDATE team_members
SET role = $3, updated_at = now()
WHERE team_id = $1 AND user_id = $2;

-- name: GetTeamMembers :many
SELECT u.id, u.email, u.name, u.username, u.avatar_url, tm.role
FROM team_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.team_id = $1
ORDER BY tm.created_at;

-- name: GetUserTeams :many
SELECT t.id, t.name, t.description, t.avatar_url, tm.role
FROM teams t
JOIN team_members tm ON t.id = tm.team_id
WHERE tm.user_id = $1
ORDER BY t.name;

-- name: CheckTeamMembership :one
SELECT EXISTS (
  SELECT 1 FROM team_members
  WHERE team_id = $1 AND user_id = $2
) AS is_member;

-- name: GetTeamMemberRole :one
SELECT role
FROM team_members
WHERE team_id = $1 AND user_id = $2;

--------------------------------------------------------
-- Projects
-- name: CreateProject :one
INSERT INTO projects (name, description, owner_id, team_id, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, description, owner_id, team_id, status, created_at, updated_at;

-- name: GetUserProjects :many
SELECT id, name, description, owner_id, team_id, status, created_at, updated_at
FROM projects
WHERE owner_id = $1
ORDER BY updated_at DESC;

-- name: GetProjectByID :one
SELECT id, name, description, owner_id, team_id, status, created_at, updated_at
FROM projects
WHERE id = $1;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;

-- name: UpdateProjectDetails :exec
UPDATE projects
SET 
  name = COALESCE($2, name),
  description = COALESCE($3, description),
  status = COALESCE($4, status),
  team_id = COALESCE($5, team_id),
  updated_at = now()
WHERE id = $1;

-- name: GetTeamProjects :many
SELECT id, name, description, owner_id, status, created_at, updated_at
FROM projects
WHERE team_id = $1
ORDER BY updated_at DESC;

-- name: GetProjectsByStatus :many
SELECT id, name, description, owner_id, team_id, created_at, updated_at
FROM projects
WHERE status = $1
ORDER BY updated_at DESC
LIMIT $2 OFFSET $3;

-- name: GetProjectStats :one
SELECT
  (SELECT COUNT(*) FROM issues WHERE issues.project_id = $1) AS total_issues,
  (SELECT COUNT(*) FROM issues WHERE issues.project_id = $1 AND issues.status = 'open') AS open_issues,
  (SELECT COUNT(*) FROM issues WHERE issues.project_id = $1 AND issues.status = 'in_progress') AS in_progress_issues,
  (SELECT COUNT(*) FROM issues WHERE issues.project_id = $1 AND issues.status = 'closed') AS closed_issues,
  (SELECT COUNT(*) FROM tasks WHERE tasks.project_id = $1) AS total_tasks,
  (SELECT COUNT(*) FROM tasks WHERE tasks.project_id = $1 AND tasks.status = 'todo') AS todo_tasks,
  (SELECT COUNT(*) FROM tasks WHERE tasks.project_id = $1 AND tasks.status = 'in_progress') AS in_progress_tasks,
  (SELECT COUNT(*) FROM tasks WHERE tasks.project_id = $1 AND tasks.status = 'done') AS done_tasks;

--------------------------------------------------------
-- Issues
-- name: CreateIssue :one
INSERT INTO issues (project_id, title, description, status, reporter_id, assignee_id, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, project_id, title, description, status, reporter_id, assignee_id, due_date, created_at, updated_at;

-- name: GetProjectIssues :many
SELECT id, title, description, status, reporter_id, assignee_id, due_date, created_at, updated_at
FROM issues
WHERE project_id = $1
ORDER BY created_at DESC;

-- name: UpdateIssueStatus :exec
UPDATE issues
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteIssue :exec
DELETE FROM issues WHERE id = $1;

-- name: GetIssuesAssignedToUser :many
SELECT i.id, i.project_id, i.title, i.description, i.status, i.reporter_id, i.due_date, 
       i.created_at, i.updated_at, p.name AS project_name
FROM issues i
JOIN projects p ON i.project_id = p.id
WHERE i.assignee_id = $1
ORDER BY i.due_date ASC NULLS LAST, i.created_at DESC;

-- name: UpdateIssueDetails :exec
UPDATE issues
SET 
  title = COALESCE($2, title),
  description = COALESCE($3, description),
  status = COALESCE($4, status),
  assignee_id = COALESCE($5, assignee_id),
  due_date = COALESCE($6, due_date),
  updated_at = now()
WHERE id = $1;

-- name: GetIssueByID :one
SELECT id, project_id, title, description, status, reporter_id, assignee_id, due_date, created_at, updated_at
FROM issues
WHERE id = $1;

-- name: GetIssuesByStatus :many
SELECT issues.id, issues.project_id, issues.title, issues.description, issues.reporter_id, 
       issues.assignee_id, issues.due_date, issues.created_at, issues.updated_at
FROM issues
WHERE issues.project_id = $1 AND issues.status = $2
ORDER BY issues.created_at DESC;

-- name: GetRecentIssues :many
SELECT i.id, i.project_id, i.title, i.status, i.due_date, p.name AS project_name
FROM issues i
JOIN projects p ON i.project_id = p.id
WHERE i.project_id IN (
    SELECT id FROM projects WHERE projects.owner_id = $1
    UNION
    SELECT p2.id FROM projects p2
    JOIN teams t ON p2.team_id = t.id
    JOIN team_members tm ON t.id = tm.team_id
    WHERE tm.user_id = $1
)
ORDER BY i.created_at DESC
LIMIT $2;

--------------------------------------------------------
-- Tasks
-- name: CreateTask :one
INSERT INTO tasks (project_id, assignee_id, title, description, status, priority, due_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, project_id, assignee_id, title, description, status, priority, due_date, created_at, updated_at;

-- name: GetUserTasks :many
SELECT t.id, t.project_id, t.title, t.description, t.status, t.priority, t.due_date, 
       t.created_at, t.updated_at, p.name AS project_name
FROM tasks t
JOIN projects p ON t.project_id = p.id
WHERE t.assignee_id = $1
ORDER BY t.due_date ASC NULLS LAST, t.priority DESC, t.created_at DESC;

-- name: UpdateTaskStatus :exec
UPDATE tasks
SET status = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteTask :exec
DELETE FROM tasks WHERE id = $1;

-- name: GetProjectTasks :many
SELECT id, assignee_id, title, description, status, priority, due_date, created_at, updated_at
FROM tasks
WHERE project_id = $1
ORDER BY priority DESC, due_date ASC NULLS LAST, created_at DESC;

-- name: UpdateTaskDetails :exec
UPDATE tasks
SET 
  title = COALESCE($2, title),
  description = COALESCE($3, description),
  status = COALESCE($4, status),
  priority = COALESCE($5, priority),
  assignee_id = COALESCE($6, assignee_id),
  due_date = COALESCE($7, due_date),
  updated_at = now()
WHERE id = $1;

-- name: GetTaskByID :one
SELECT id, project_id, assignee_id, title, description, status, priority, due_date, created_at, updated_at
FROM tasks
WHERE id = $1;

-- name: GetTasksByStatus :many
SELECT id, project_id, assignee_id, title, description, priority, due_date, created_at, updated_at
FROM tasks
WHERE project_id = $1 AND status = $2
ORDER BY priority DESC, due_date ASC NULLS LAST;

-- name: GetOverdueTasks :many
SELECT t.id, t.project_id, t.assignee_id, t.title, t.status, t.priority, t.due_date, 
       p.name AS project_name
FROM tasks t
JOIN projects p ON t.project_id = p.id
WHERE t.due_date < now() AND t.status != 'done' AND t.assignee_id = $1
ORDER BY t.due_date ASC;

--------------------------------------------------------
-- Comments
-- name: CreateComment :one
INSERT INTO comments (content, user_id, issue_id, task_id)
VALUES ($1, $2, $3, $4)
RETURNING id, content, user_id, issue_id, task_id, created_at, updated_at;

-- name: GetCommentsByIssue :many
SELECT c.id, c.content, c.user_id, c.created_at, c.updated_at, 
       u.name AS user_name, u.username, u.avatar_url
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.issue_id = $1
ORDER BY c.created_at ASC;

-- name: GetCommentsByTask :many
SELECT c.id, c.content, c.user_id, c.created_at, c.updated_at, 
       u.name AS user_name, u.username, u.avatar_url
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.task_id = $1
ORDER BY c.created_at ASC;

-- name: UpdateCommentContent :exec
UPDATE comments
SET content = $2, updated_at = now()
WHERE id = $1 AND user_id = $3;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1 AND user_id = $2;

-- name: GetCommentByID :one
SELECT id, content, user_id, issue_id, task_id, created_at, updated_at
FROM comments
WHERE id = $1;

-- name: GetRecentComments :many
SELECT c.id, c.content, c.user_id, c.issue_id, c.task_id, c.created_at,
       u.name AS user_name, u.username
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.issue_id IN (
    SELECT id FROM issues WHERE issues.assignee_id = $1
) OR c.task_id IN (
    SELECT id FROM tasks WHERE tasks.assignee_id = $1
)
ORDER BY c.created_at DESC
LIMIT $2;

--------------------------------------------------------
-- Dashboard Queries
-- name: GetUserDashboardStats :one
SELECT 
  (SELECT COUNT(*) FROM projects WHERE owner_id = $1) AS owned_projects,
  (SELECT COUNT(*) FROM issues WHERE assignee_id = $1) AS assigned_issues,
  (SELECT COUNT(*) FROM issues WHERE assignee_id = $1 AND status != 'closed') AS open_issues,
  (SELECT COUNT(*) FROM tasks WHERE assignee_id = $1) AS assigned_tasks,
  (SELECT COUNT(*) FROM tasks WHERE assignee_id = $1 AND status != 'done') AS pending_tasks,
  (SELECT COUNT(*) FROM tasks WHERE assignee_id = $1 AND due_date < now() AND status != 'done') AS overdue_tasks;

-- name: GetUserActivityFeed :many
WITH user_activities AS (
  -- Projects created
  SELECT 'project_created' AS activity_type, p.id AS entity_id, p.name AS entity_name,
         null::uuid AS related_entity_id, null AS related_entity_name,
         p.created_at AS activity_time
  FROM projects p
  WHERE p.owner_id = $1
  
  UNION ALL
  
  -- Issues created
  SELECT 'issue_created' AS activity_type, i.id AS entity_id, i.title AS entity_name,
         i.project_id AS related_entity_id, p.name AS related_entity_name,
         i.created_at AS activity_time
  FROM issues i
  JOIN projects p ON i.project_id = p.id
  WHERE i.reporter_id = $1
  
  UNION ALL
  
  -- Comments created
  SELECT 'comment_created' AS activity_type, c.id AS entity_id, c.content AS entity_name,
         COALESCE(c.issue_id, c.task_id) AS related_entity_id,
         COALESCE(i.title, t.title) AS related_entity_name,
         c.created_at AS activity_time
  FROM comments c
  LEFT JOIN issues i ON c.issue_id = i.id
  LEFT JOIN tasks t ON c.task_id = t.id
  WHERE c.user_id = $1
  
  UNION ALL
  
  -- Tasks created or updated
  SELECT 'task_updated' AS activity_type, t.id AS entity_id, t.title AS entity_name,
         t.project_id AS related_entity_id, p.name AS related_entity_name,
         t.updated_at AS activity_time
  FROM tasks t
  JOIN projects p ON t.project_id = p.id
  WHERE t.assignee_id = $1 AND t.updated_at > t.created_at
)
SELECT * FROM user_activities
ORDER BY activity_time DESC
LIMIT $2;

-- name: SearchEntities :many
WITH search_results AS (
  -- Projects
  SELECT 'project' AS entity_type, p.id AS entity_id, p.name AS entity_name,
         p.description AS entity_description, p.created_at,
         p.owner_id AS user_id, null::uuid AS parent_id
  FROM projects p
  WHERE (p.owner_id = $1 OR p.team_id IN (SELECT team_id FROM team_members WHERE user_id = $1))
    AND (p.name ILIKE '%' || $2 || '%' OR p.description ILIKE '%' || $2 || '%')
  
  UNION ALL
  
  -- Issues
  SELECT 'issue' AS entity_type, i.id AS entity_id, i.title AS entity_name,
         i.description AS entity_description, i.created_at,
         i.reporter_id AS user_id, i.project_id AS parent_id
  FROM issues i
  JOIN projects p ON i.project_id = p.id
  WHERE (p.owner_id = $1 OR p.team_id IN (SELECT team_id FROM team_members WHERE user_id = $1))
    AND (i.title ILIKE '%' || $2 || '%' OR i.description ILIKE '%' || $2 || '%')
  
  UNION ALL
  
  -- Tasks
  SELECT 'task' AS entity_type, t.id AS entity_id, t.title AS entity_name,
         t.description AS entity_description, t.created_at,
         t.assignee_id AS user_id, t.project_id AS parent_id
  FROM tasks t
  JOIN projects p ON t.project_id = p.id
  WHERE (p.owner_id = $1 OR p.team_id IN (SELECT team_id FROM team_members WHERE user_id = $1))
    AND (t.title ILIKE '%' || $2 || '%' OR t.description ILIKE '%' || $2 || '%')
)
SELECT * FROM search_results
ORDER BY created_at DESC
LIMIT $3;