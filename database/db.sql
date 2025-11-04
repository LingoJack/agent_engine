CREATE TABLE IF NOT EXISTS t_tool (
                                      id INTEGER PRIMARY KEY AUTOINCREMENT,
                                      tool_id TEXT NOT NULL,
                                      tool_name TEXT,
                                      description TEXT,
                                      document TEXT,
                                      example TEXT,
                                      status TEXT,
                                      create_time DATETIME DEFAULT CURRENT_TIMESTAMP,
                                      update_time DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tool_id ON t_tool (tool_id);