package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Logger define a interface para logging estruturado.
// A aplicação (Handler, Service) deve depender apenas desta interface.
type Logger interface {
	Debug(msg string, fields map[string]interface{})
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error)
	Fatal(msg string, err error)
}

// LogEntry define a estrutura de um log para garantir o formato JSON.
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// SimpleLogger é uma implementação concreta da interface Logger
// que usa o pacote log nativo, mas com output JSON estruturado.
type SimpleLogger struct {
	logLevel string // e.g., "debug", "info", "error"
}

// NewLogger cria e retorna uma nova instância do Logger.
// Esta função é chamada no main.go.
func NewLogger(level string) Logger {
	// Configura o logger padrão do Go para não incluir prefixos duplicados.
	log.SetFlags(0)
	return &SimpleLogger{logLevel: level}
}

// logf formata a entrada como JSON e a escreve na saída padrão.
func (l *SimpleLogger) logf(level, msg string, fields map[string]interface{}, err error) {
	// Apenas registra logs se o nível for apropriado (lógica simplificada)
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		Message:   msg,
	}

	if fields != nil {
		entry.Fields = fields
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Serializa a entrada para JSON
	jsonBytes, _ := json.Marshal(entry)

	// Escreve no stderr ou stdout (dependendo do nível, mas usando log.Println para simplicidade)
	log.Println(string(jsonBytes))

	// Se for Fatal, o programa deve ser encerrado
	if level == "FATAL" {
		os.Exit(1)
	}
}

// shouldLog implementa uma lógica básica de nível de log.
func (l *SimpleLogger) shouldLog(level string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"error": 2,
		"fatal": 3,
	}

	currentLevel, ok := levels[l.logLevel]
	if !ok {
		currentLevel = 1 // Default to info
	}

	targetLevel, ok := levels[level]
	if !ok {
		return false
	}

	return targetLevel >= currentLevel
}

// Implementações da Interface Logger

func (l *SimpleLogger) Debug(msg string, fields map[string]interface{}) {
	l.logf("DEBUG", msg, fields, nil)
}

func (l *SimpleLogger) Info(msg string, fields map[string]interface{}) {
	l.logf("INFO", msg, fields, nil)
}

func (l *SimpleLogger) Error(msg string, err error) {
	l.logf("ERROR", msg, nil, err)
}

func (l *SimpleLogger) Fatal(msg string, err error) {
	l.logf("FATAL", msg, nil, err)
}
