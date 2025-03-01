-- Comments migration file
-- This file adds the comments table and related indexes/triggers

-- Comments Table
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    CHECK (
        (issue_id IS NOT NULL AND task_id IS NULL) OR
        (issue_id IS NULL AND task_id IS NOT NULL)
    )
);

-- Create indexes for the comments table
CREATE INDEX idx_comments_issue ON comments(issue_id);
CREATE INDEX idx_comments_task ON comments(task_id);
CREATE INDEX idx_comments_user ON comments(user_id);

-- Create trigger for the comments table
CREATE TRIGGER update_comments_updated_at
BEFORE UPDATE ON comments
FOR EACH ROW
EXECUTE FUNCTION update_timestamp(); 