package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const runtimeCallerSkip = 4

var currentDir = "."

type logEntry struct {
	Date string `json:"date"`
	Time string `json:"time"`

	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`

	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields"`
}
type Handler struct {
	slog.Handler
	lConsole *log.Logger
	lFile    *log.Logger
}

func init() {
	dir, err := os.Getwd()
	if err != nil {
		currentDir = "."
	}
	currentDir = dir
}

func NewHandler(debug bool) *Handler {
	fileName := fmt.Sprintf("storage/logs/app_%s.log", time.Now().Format("2006-01-02"))
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Ошибка открытия файла: %v", err)
	}
	multiWriterFile := io.MultiWriter(file)

	if debug {
		return &Handler{
			Handler: slog.NewJSONHandler(multiWriterFile, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
			lConsole: log.New(os.Stdout, "", 0),
			lFile:    log.New(multiWriterFile, "", 0),
		}
	}
	return &Handler{
		Handler: slog.NewTextHandler(multiWriterFile, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
		lConsole: log.New(os.Stdout, "", 0),
		lFile:    log.New(multiWriterFile, "", 0),
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	err := h.console(r)
	if err != nil {
		return err
	}
	err = h.file(r)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) console(r slog.Record) error {
	level := r.Level.String() + ":"
	switch r.Level {
	case slog.LevelDebug:
		level = color.MagentaString(level)
	case slog.LevelInfo:
		level = color.BlueString(level)
	case slog.LevelWarn:
		level = color.YellowString(level)
	case slog.LevelError:
		level = color.RedString(level)
	}
	fields := make(map[string]interface{}, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()
		return true
	})
	b, err := json.MarshalIndent(fields, "", "  ")
	if err != nil {
		return err
	}
	timeStr := r.Time.Format("[15:05:05]")
	_, file, line, ok := runtime.Caller(runtimeCallerSkip)
	if ok {
		relPath, err := filepath.Rel(currentDir, file)
		if err != nil {
			relPath = file
		}
		h.lConsole.Printf("%s %s %s:%s [ %s ] %s \n", color.GreenString(timeStr), level, color.HiCyanString(relPath), color.HiCyanString(strconv.Itoa(line)), color.CyanString(r.Message), color.HiWhiteString(string(b)))
	} else {
		h.lConsole.Println(color.GreenString(timeStr), level, color.CyanString(r.Message), color.WhiteString(string(b)))
	}

	return nil
}

func (h *Handler) file(r slog.Record) error {
	fields := make(map[string]interface{}, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields[a.Key] = a.Value.Any()
		return true
	})
	date := r.Time.Format("2006-01-02")
	timeStr := r.Time.Format("15:05:05")

	pc, file, line, ok := runtime.Caller(runtimeCallerSkip)
	lg := logEntry{}
	if ok {
		relPath, err := filepath.Rel(currentDir, file)
		if err != nil {
			relPath = file
		}
		relPath = filepath.ToSlash(relPath)
		funcName := runtime.FuncForPC(pc).Name()
		lg = logEntry{
			Date:     date,
			Time:     timeStr,
			Level:    r.Level.String(),
			Line:     line,
			Message:  r.Message,
			File:     relPath,
			Function: funcName,
			Fields:   fields,
		}
	} else {
		lg = logEntry{
			Date:     date,
			Time:     timeStr,
			Level:    r.Level.String(),
			Line:     line,
			Message:  r.Message,
			File:     file,
			Function: "",
			Fields:   fields,
		}
	}

	jsonLog, err := json.MarshalIndent(lg, "", "  ")
	if err != nil {
		return err
	}
	h.lFile.Println(string(jsonLog))
	return nil
}
