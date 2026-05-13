// Package session manages open document state for the editor.
package session

import (
	"sync"

	"github.com/iceisfun/golua/v1/ast"
	"github.com/iceisfun/golua/v1/examples/editor_advanced/language"
	"github.com/iceisfun/golua/v1/parser"
)

// Document holds the current state of an open file.
type Document struct {
	URI     string
	Text    string
	Version int
	AST     *ast.Block           // from ParsePartial, never nil
	Symbols *language.SymbolTable // scope analysis result
}

// Manager tracks open documents, keyed by URI.
type Manager struct {
	mu   sync.RWMutex
	docs map[string]*Document
}

// NewManager creates a new document manager.
func NewManager() *Manager {
	return &Manager{docs: make(map[string]*Document)}
}

// Open registers a new document.
func (m *Manager) Open(uri string, version int, text string) *Document {
	doc := &Document{URI: uri, Version: version, Text: text}
	analyze(doc)
	m.mu.Lock()
	m.docs[uri] = doc
	m.mu.Unlock()
	return doc
}

// Update re-analyzes a document after a content change.
func (m *Manager) Update(uri string, version int, text string) *Document {
	m.mu.Lock()
	doc, ok := m.docs[uri]
	if !ok {
		doc = &Document{URI: uri}
		m.docs[uri] = doc
	}
	doc.Text = text
	doc.Version = version
	m.mu.Unlock()

	analyze(doc)
	return doc
}

// Get returns a document by URI, or nil if not found.
func (m *Manager) Get(uri string) *Document {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.docs[uri]
}

// Close removes a document.
func (m *Manager) Close(uri string) {
	m.mu.Lock()
	delete(m.docs, uri)
	m.mu.Unlock()
}

func analyze(doc *Document) {
	block, _ := parser.ParsePartial("editor", doc.Text)
	doc.AST = block
	doc.Symbols = language.Analyze(block, doc.Text)
}
