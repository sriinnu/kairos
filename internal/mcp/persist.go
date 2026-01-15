package mcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kairos/internal/storage"
)

// Memory represents a stored memory
type Memory struct {
	ID        int64     `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func initMemoriesTable(db *storage.Database) {
	query := `
	CREATE TABLE IF NOT EXISTS memories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL,
		category TEXT,
		tags TEXT,
		created_at TEXT,
		updated_at TEXT
	)`
	db.Exec(query)
}

func handlePersist(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	initMemoriesTable(db)

	action, _ := args["action"].(string)

	switch action {
	case "store":
		return persistStore(db, args)
	case "retrieve":
		return persistRetrieve(db, args)
	case "search":
		return persistSearch(db, args)
	case "list":
		return persistList(db, args)
	case "delete":
		return persistDelete(db, args)
	case "update":
		return persistUpdate(db, args)
	case "cleanup":
		return persistCleanup(db, args)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func persistStore(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)
	category, _ := args["category"].(string)

	tagsRaw, _ := args["tags"].([]interface{})
	var tags []string
	for _, t := range tagsRaw {
		if s, ok := t.(string); ok {
			tags = append(tags, s)
		}
	}

	tagsJSON, _ := json.Marshal(tags)
	now := time.Now().Format(time.RFC3339)

	// Check if exists
	var existingCreated string
	db.QueryRow("SELECT created_at FROM memories WHERE key = ?", key).Scan(&existingCreated)

	if existingCreated != "" {
		// Update
		db.Exec("UPDATE memories SET value=?, category=?, tags=?, updated_at=? WHERE key=?",
			value, category, string(tagsJSON), now, key)
		return map[string]interface{}{
			"action":     "update",
			"key":        key,
			"updated_at": now,
		}, nil
	}

	// Insert
	db.Exec("INSERT INTO memories (key, value, category, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		key, value, category, string(tagsJSON), now, now)

	return map[string]interface{}{
		"action":     "store",
		"key":        key,
		"stored_at":  now,
		"category":   category,
		"tags":       tags,
	}, nil
}

func persistRetrieve(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	key, _ := args["key"].(string)

	var memory Memory
	var tags string

	err := db.QueryRow(
		"SELECT key, value, category, tags, created_at, updated_at FROM memories WHERE key = ?",
		key,
	).Scan(&memory.Key, &memory.Value, &memory.Category, &tags, &memory.CreatedAt, &memory.UpdatedAt)

	if err != nil {
		return map[string]interface{}{
			"found": false,
			"key":   key,
		}, nil
	}

	json.Unmarshal([]byte(tags), &memory.Tags)
	return map[string]interface{}{
		"found":     true,
		"key":       memory.Key,
		"value":     memory.Value,
		"category":  memory.Category,
		"tags":      memory.Tags,
		"created":   memory.CreatedAt,
		"updated":   memory.UpdatedAt,
	}, nil
}

func persistSearch(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	query, _ := args["query"].(string)
	category, _ := args["category"].(string)

	memories, err := getAllMemories(db)
	if err != nil {
		return nil, err
	}

	var results []Memory
	for _, m := range memories {
		match := false
		if query != "" {
			if strings.Contains(strings.ToLower(m.Key), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(m.Value), strings.ToLower(query)) {
				match = true
			}
			for _, tag := range m.Tags {
				if strings.Contains(strings.ToLower(tag), strings.ToLower(query)) {
					match = true
					break
				}
			}
		}
		if category != "" && m.Category == category {
			match = true
		}
		if match {
			results = append(results, m)
		}
	}

	return map[string]interface{}{
		"count":   len(results),
		"results": results,
	}, nil
}

func persistList(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	category, _ := args["category"].(string)

	memories, err := getAllMemories(db)
	if err != nil {
		return nil, err
	}

	if category != "" {
		var filtered []Memory
		for _, m := range memories {
			if m.Category == category {
				filtered = append(filtered, m)
			}
		}
		memories = filtered
	}

	return map[string]interface{}{
		"count":     len(memories),
		"memories": memories,
	}, nil
}

func persistDelete(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	key, _ := args["key"].(string)
	db.Exec("DELETE FROM memories WHERE key = ?", key)

	return map[string]interface{}{
		"deleted":    true,
		"key":        key,
		"deleted_at": time.Now().Format(time.RFC3339),
	}, nil
}

func persistUpdate(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	key, _ := args["key"].(string)
	value, _ := args["value"].(string)
	category, _ := args["category"].(string)

	tagsRaw, _ := args["tags"].([]interface{})
	var tags []string
	for _, t := range tagsRaw {
		if s, ok := t.(string); ok {
			tags = append(tags, s)
		}
	}

	tagsJSON, _ := json.Marshal(tags)
	now := time.Now().Format(time.RFC3339)

	db.Exec("UPDATE memories SET value=?, category=?, tags=?, updated_at=? WHERE key=?",
		value, category, string(tagsJSON), now, key)

	return map[string]interface{}{
		"updated":    true,
		"key":        key,
		"updated_at": now,
	}, nil
}

func persistCleanup(db *storage.Database, args map[string]interface{}) (interface{}, error) {
	olderThan, _ := args["older_than"].(string)

	var query string
	var params []interface{}

	if olderThan != "" {
		t, err := time.Parse("2006-01-02", olderThan)
		if err == nil {
			query = "DELETE FROM memories WHERE created_at < ?"
			params = append(params, t.Format(time.RFC3339))
		}
	}

	if query == "" {
		// Delete all without created_at (cleanup orphaned)
		db.Exec("DELETE FROM memories WHERE key = ''")
	}

	_ = db.Exec(query, params...)
	return map[string]interface{}{
		"cleaned":     true,
		"cleaned_at":  time.Now().Format(time.RFC3339),
	}, nil
}

func getAllMemories(db *storage.Database) ([]Memory, error) {
	rows, err := db.Query(
		"SELECT key, value, category, tags, created_at, updated_at FROM memories ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var memory Memory
		var tags string
		rows.Scan(&memory.Key, &memory.Value, &memory.Category, &tags, &memory.CreatedAt, &memory.UpdatedAt)
		json.Unmarshal([]byte(tags), &memory.Tags)
		memories = append(memories, memory)
	}

	return memories, rows.Err()
}
