CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY, 
    name TEXT, 
    avatar TEXT, 
    
    is_group BOOLEAN DEFAULT FALSE,
    last_message_id BIGINT, 
    last_message_at TIMESTAMP NOT NULL DEFAULT now(),
    
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now() 
);

CREATE INDEX IF NOT EXISTS idx_conversations_last_msg ON conversations(last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_conversations_is_group ON conversations(is_group);

CREATE TABLE IF NOT EXISTS participants (
    conversation_id BIGINT NOT NULL,
    user_id VARCHAR(25) NOT NULL,
    
    role VARCHAR(20) DEFAULT 'member', 
    joined_at TIMESTAMP NOT NULL DEFAULT now(),
    
    last_read_message_id BIGINT, 

    PRIMARY KEY (conversation_id, user_id),
    CONSTRAINT fk_participant_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_participants_user ON participants(user_id);


CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY, 
    conversation_id BIGINT NOT NULL,
    sender_id VARCHAR(25) NOT NULL,
    
    content TEXT, 
    type VARCHAR(20) DEFAULT 'text', 
    
    reply_to_id BIGINT, 
    
    is_deleted BOOLEAN DEFAULT FALSE, 
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    
    CONSTRAINT fk_msg_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at DESC);

