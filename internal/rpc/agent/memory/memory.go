package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cloudwego/eino/schema"
)

//examples中的简易内存示例，自己修修改改了些，还不错
//和我之前的memory实现差不多欸嘿嘿

// 默认设置
func GetDefaultMemory() *SimpleMemory {
	return NewSimpleMemory(SimpleMemoryConfig{
		Dir:           "data/memory",
		MaxWindowSize: 6,
	})
}

// 配置
type SimpleMemoryConfig struct {
	Dir           string //存储路径
	MaxWindowSize int    //最大返回窗口大小
}

type SimpleMemory struct {
	mu            sync.Mutex
	dir           string
	maxWindowSize int
	conversations map[string]*Conversation
}

type Conversation struct {
	mu sync.Mutex

	ID string `json:"id"`
	//每一条对话是一个message
	Messages []*schema.Message `json:"messages"`

	filePath      string
	maxWindowSize int
}

// 创建存储目录、对话map
func NewSimpleMemory(cfg SimpleMemoryConfig) *SimpleMemory {
	if err := os.MkdirAll(cfg.Dir, 0755); err != nil {
		return nil
	}

	return &SimpleMemory{
		dir:           cfg.Dir,
		maxWindowSize: cfg.MaxWindowSize,
		conversations: make(map[string]*Conversation),
	}
}

// 获取对话
func (m *SimpleMemory) GetConversation(id string, createIfNotExist bool) *Conversation {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.conversations[id]

	filePath := filepath.Join(m.dir, id+".jsonl")
	if !ok {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			if createIfNotExist {
				if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
					return nil
				}
				m.conversations[id] = &Conversation{
					ID:            id,
					Messages:      make([]*schema.Message, 0),
					filePath:      filePath,
					maxWindowSize: m.maxWindowSize,
				}
			}
		}

		con := &Conversation{
			ID:            id,
			Messages:      make([]*schema.Message, 0),
			filePath:      filePath,
			maxWindowSize: m.maxWindowSize,
		}
		con.load()
		m.conversations[id] = con
	}

	return m.conversations[id]
}

// 列出所有对话文件
func (m *SimpleMemory) ListConversations() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	files, err := os.ReadDir(m.dir)
	if err != nil {
		return nil
	}

	ids := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		ids = append(ids, strings.TrimSuffix(file.Name(), ".jsonl"))
	}

	return ids
}

// 删除对话
func (m *SimpleMemory) DeleteConversation(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	filePath := filepath.Join(m.dir, id+".jsonl")
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	delete(m.conversations, id)
	return nil
}

func (c *Conversation) Append(msg *schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Messages = append(c.Messages, msg)

	c.save(msg)
}

func (c *Conversation) GetFullMessages() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Messages
}

// get messages with max window size
func (c *Conversation) GetMessages() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.Messages) > c.maxWindowSize {
		return c.Messages[len(c.Messages)-c.maxWindowSize:]
	}

	return c.Messages
}

func (c *Conversation) load() error {
	reader, err := os.Open(c.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		var msg schema.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}
		c.Messages = append(c.Messages, &msg)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func (c *Conversation) save(msg *schema.Message) {
	str, _ := json.Marshal(msg)

	// Append to file
	f, err := os.OpenFile(c.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(str)
	f.WriteString("\n")
}
