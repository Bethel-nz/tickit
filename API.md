# Tickit API Documentation

## Authentication

All protected endpoints require a JWT token in the Authorization header:

```
Authorization: Bearer <your_jwt_token>
```

## User Management

### Register User

```http
POST /users/register
Content-Type: application/json

{
    "email": "user@example.com",
    "password": "securepassword",
    "name": "John Doe",
    "username": "johndoe"
}
```

### Login

```http
POST /users/login
Content-Type: application/json

{
    "email": "user@example.com",
    "password": "securepassword"
}
```

### Forgot Password

```http
POST /users/forgot-password
Content-Type: application/json

{
    "email": "user@example.com"
}
```

### Reset Password

```http
POST /users/reset-password/{token}
Content-Type: application/json

{
    "new_password": "newsecurepassword"
}
```

### Get User Profile

```http
GET /users/me
Authorization: Bearer <token>
```

### Update User Profile

```http
PUT /users/me
Authorization: Bearer <token>
Content-Type: application/json

{
    "name": "Updated Name",
    "username": "updatedusername"
}
```

### Change Password

```http
POST /users/change-password
Authorization: Bearer <token>
Content-Type: application/json

{
    "current_password": "currentpassword",
    "new_password": "newpassword"
}
```

### Delete Account

```http
DELETE /users/me
Authorization: Bearer <token>
```

## Projects

### List Projects

```http
GET /projects
Authorization: Bearer <token>
```

### Create Project

```http
POST /projects
Authorization: Bearer <token>
Content-Type: application/json

{
    "name": "Project Name",
    "description": "Project Description"
}
```

### Get Project

```http
GET /projects/{id}
Authorization: Bearer <token>
```

### Update Project

```http
PUT /projects/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
    "name": "Updated Project Name",
    "description": "Updated Description"
}
```

### Delete Project

```http
DELETE /projects/{id}
Authorization: Bearer <token>
```

## Tickets

### List Tickets

```http
GET /projects/{project_id}/tickets
Authorization: Bearer <token>
```

### Create Ticket

```http
POST /projects/{project_id}/tickets
Authorization: Bearer <token>
Content-Type: application/json

{
    "title": "Ticket Title",
    "description": "Ticket Description",
    "priority": "high",
    "status": "open"
}
```

### Get Ticket

```http
GET /projects/{project_id}/tickets/{id}
Authorization: Bearer <token>
```

### Update Ticket

```http
PUT /projects/{project_id}/tickets/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
    "title": "Updated Title",
    "description": "Updated Description",
    "priority": "medium",
    "status": "in_progress"
}
```

### Delete Ticket

```http
DELETE /projects/{project_id}/tickets/{id}
Authorization: Bearer <token>
```

### Assign Ticket

```http
POST /projects/{project_id}/tickets/{id}/assign
Authorization: Bearer <token>
Content-Type: application/json

{
    "assignee_id": "user-uuid"
}
```

## Comments

### List Comments

```http
GET /projects/{project_id}/tickets/{ticket_id}/comments
Authorization: Bearer <token>
```

### Create Comment

```http
POST /projects/{project_id}/tickets/{ticket_id}/comments
Authorization: Bearer <token>
Content-Type: application/json

{
    "content": "Comment content"
}
```

### Update Comment

```http
PUT /projects/{project_id}/tickets/{ticket_id}/comments/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
    "content": "Updated comment content"
}
```

### Delete Comment

```http
DELETE /projects/{project_id}/tickets/{ticket_id}/comments/{id}
Authorization: Bearer <token>
```

## Task Comments

### List Task Comments

```http
GET /projects/{project_id}/tasks/{task_id}/comments
Authorization: Bearer <token>
```

### Create Task Comment

```http
POST /projects/{project_id}/tasks/{task_id}/comments
Authorization: Bearer <token>
Content-Type: application/json

{
    "content": "Task comment content"
}
```

## Search

### Search Entities

```http
GET /search?q=search_term
Authorization: Bearer <token>
```

## Health Check

### Check API Status

```http
GET /health
```
