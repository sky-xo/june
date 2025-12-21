package repo

import (
	"database/sql"
	"encoding/json"
)

type Message struct {
	ID            string
	FromID        string
	Type          string
	Content       string
	MentionsJSON  string
	RequiresHuman bool
	ReadByJSON    string
}

type MessageFilter struct {
	Type     string
	FromID   string
	Limit    int
	Mention  string
	ReaderID string
}

func CreateMessage(db *sql.DB, m Message) error {
	_, err := db.Exec(`INSERT INTO messages (id, from_id, type, content, mentions, requires_human, read_by) VALUES (?, ?, ?, ?, ?, ?, ?)`, m.ID, m.FromID, m.Type, m.Content, m.MentionsJSON, m.RequiresHuman, m.ReadByJSON)
	return err
}

func ListMessages(db *sql.DB, f MessageFilter) ([]Message, error) {
	query := `SELECT id, from_id, type, content, mentions, requires_human, read_by FROM messages`
	var args []interface{}
	where := ""
	if f.Type != "" {
		where = appendWhere(where, "type = ?")
		args = append(args, f.Type)
	}
	if f.FromID != "" {
		where = appendWhere(where, "from_id = ?")
		args = append(args, f.FromID)
	}
	if where != "" {
		query += " WHERE " + where
	}
	query += " ORDER BY created_at ASC"
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.FromID, &m.Type, &m.Content, &m.MentionsJSON, &m.RequiresHuman, &m.ReadByJSON); err != nil {
			return nil, err
		}
		if f.Mention != "" && !mentionsContain(m.MentionsJSON, f.Mention) {
			continue
		}
		if f.ReaderID != "" && readByContains(m.ReadByJSON, f.ReaderID) {
			continue
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func appendWhere(existing, clause string) string {
	if existing == "" {
		return clause
	}
	return existing + " AND " + clause
}

func mentionsContain(mentionsJSON, mention string) bool {
	var items []string
	if err := json.Unmarshal([]byte(mentionsJSON), &items); err != nil {
		return false
	}
	for _, item := range items {
		if item == mention {
			return true
		}
	}
	return false
}

func readByContains(readByJSON, reader string) bool {
	var items []string
	if err := json.Unmarshal([]byte(readByJSON), &items); err != nil {
		return false
	}
	for _, item := range items {
		if item == reader {
			return true
		}
	}
	return false
}
