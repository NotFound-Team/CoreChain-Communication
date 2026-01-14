
CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    host_id VARCHAR(255) NOT NULL,
    room_name VARCHAR(100) UNIQUE NOT NULL, 
    meeting_key VARCHAR(20) UNIQUE NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE, 
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_unique_active_meeting_key 
ON meetings (meeting_key) 
WHERE end_time IS NULL;


CREATE TABLE meeting_invites (
    meeting_id UUID REFERENCES meetings(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    PRIMARY KEY (meeting_id, user_id)
);

CREATE INDEX idx_invites_user ON meeting_invites(user_id);
CREATE INDEX idx_meetings_active ON meetings(is_active) WHERE is_active = true;

