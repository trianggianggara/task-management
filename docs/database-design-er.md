erDiagram
    teams ||--o{ users : "team_id"
    users ||--o{ tasks : "user_id (reporter)"
    users ||--o{ tasks : "assignee_id"
    users ||--o{ idempotency_keys : "user_id"
    tasks ||--o{ task_logs : "task_id"
    users ||--o{ task_logs : "changed_by"

    teams {
        UUID id PK
        VARCHAR20 code UK
        VARCHAR100 name
        TIMESTAMPTZ created_at
    }

    users {
        UUID id PK
        VARCHAR255 email UK
        VARCHAR255 password_hash
        VARCHAR100 name
        UUID team_id FK
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    tasks {
        UUID id PK
        UUID user_id FK "reporter"
        UUID assignee_id FK "nullable"
        VARCHAR255 title
        TEXT description
        ENUM status "pending|in_progress|completed"
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
        TIMESTAMPTZ deleted_at "soft delete"
    }

    idempotency_keys {
        UUID key PK
        UUID user_id FK
        INT response_status
        JSONB response_body
        TIMESTAMPTZ created_at
        TIMESTAMPTZ expires_at "NOW+24h"
    }

    task_logs {
        UUID id PK
        UUID task_id FK "CASCADE"
        VARCHAR50 action
        JSONB old_value "assignee_id"
        JSONB new_value "assignee_id"
        UUID changed_by FK "CASCADE"
        TIMESTAMPTZ created_at
    }
