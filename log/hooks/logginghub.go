package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/samkreter/go-core/log"

	"github.com/sirupsen/logrus"
)

const (
	timeFormat               = "2006-01-02T15:04:05.000Z07:00"
	channelBufferSize        = 1024
	defaultHTTPClientTimeout = time.Second * 30
	defaultFlushInterval     = time.Second * 30
)

var defaultLevels = []logrus.Level{
	logrus.PanicLevel,
	logrus.FatalLevel,
	logrus.ErrorLevel,
	logrus.WarnLevel,
	logrus.InfoLevel,
}

// LoggingHubEntriesReq request for the log entries
type LoggingHubEntriesReq struct {
	Senders []string           `json:"senders"`
	Entries []*LoggingHubEntry `json:"entries"`
}

// LoggingHubEntry logging hub log entry
type LoggingHubEntry struct {
	Log    string            `json:"log"`
	Time   string            `json:"time"`
	Level  string            `json:"level"`
	Fields map[string]string `json:"fields"`
}

// LoggingHubHook logrus hook for the logging agent
type LoggingHubHook struct {
	config       Config
	levels       []logrus.Level
	pendingLogs  *pendingLogs
	channel      chan *LoggingHubEntry
	ignoreFields map[string]struct{}
	filters      map[string]func(interface{}) interface{}
}

// Config configuration for the logging agent hook
type Config struct {
	LoggingHubURL       string
	Senders             []string
	LogLevels           []logrus.Level
	DefaultIgnoreFields map[string]struct{}
	DefaultFilters      map[string]func(interface{}) interface{}
	BatchSizeInLines    int
	RequestSizeLimit    int
	FlushInterval       time.Duration
}

// pendingLogs allows for thread safe access to the slice of logs
type pendingLogs struct {
	sync.Mutex
	items     []*LoggingHubEntry
	totalSize int
}

func (pl *pendingLogs) appendLog(entry *LoggingHubEntry) (int, int) {
	pl.Lock()
	defer pl.Unlock()
	pl.totalSize = pl.totalSize + getSize(entry)
	pl.items = append(pl.items, entry)
	return len(pl.items), pl.totalSize
}

func (pl *pendingLogs) flush() []*LoggingHubEntry {
	pl.Lock()
	defer pl.Unlock()
	entries := pl.items[0:]
	pl.items = []*LoggingHubEntry{}
	return entries
}

// NewLoggingHubHook creates a new logging agent hook
func NewLoggingHubHook(loggingHubURL string, senders []string) (*LoggingHubHook, error) {
	return NewWithConfig(Config{
		LoggingHubURL: loggingHubURL,
		Senders:       senders,
		FlushInterval: defaultFlushInterval,
	})
}

// NewWithConfig returns initialized logrus hook by config setting.
func NewWithConfig(conf Config) (*LoggingHubHook, error) {
	if conf.LoggingHubURL == "" {
		return nil, fmt.Errorf("loggingHubURL can no be empty")
	}

	if len(conf.Senders) == 0 {
		return nil, fmt.Errorf("configuration must have at least one sender")
	}

	if conf.FlushInterval == 0 {
		conf.FlushInterval = defaultFlushInterval
	}

	hook := &LoggingHubHook{
		config:       conf,
		levels:       conf.LogLevels,
		ignoreFields: make(map[string]struct{}),
		filters:      make(map[string]func(interface{}) interface{}),
		channel:      make(chan *LoggingHubEntry, channelBufferSize),
		pendingLogs:  &pendingLogs{},
	}
	// set default values
	if len(hook.levels) == 0 {
		hook.levels = defaultLevels
	}

	for k, v := range conf.DefaultIgnoreFields {
		hook.ignoreFields[k] = v
	}
	for k, v := range conf.DefaultFilters {
		hook.filters[k] = v
	}

	go hook.Run()

	return hook, nil
}

// Levels returns logging level to fire this hook.
func (hook *LoggingHubHook) Levels() []logrus.Level {
	return hook.levels
}

// SetLevels sets logging level to fire this hook.
func (hook *LoggingHubHook) SetLevels(levels []logrus.Level) {
	hook.levels = levels
}

// AddIgnore adds field name to ignore.
func (hook *LoggingHubHook) AddIgnore(name string) {
	hook.ignoreFields[name] = struct{}{}
}

// AddFilter adds a custom filter function.
func (hook *LoggingHubHook) AddFilter(name string, fn func(interface{}) interface{}) {
	hook.filters[name] = fn
}

// Fire is invoked by logrus and sends log to fluentd logger.
func (hook *LoggingHubHook) Fire(entry *logrus.Entry) error {
	loggingHubEntry := &LoggingHubEntry{
		Log:    entry.Message,
		Level:  entry.Level.String(),
		Time:   entry.Time.UTC().Format(timeFormat),
		Fields: make(map[string]string),
	}

	for k, v := range entry.Data {
		if _, ok := hook.ignoreFields[k]; ok {
			continue
		}

		if fn, ok := hook.filters[k]; ok {
			v = fn(v)
		} else {
			v = formatData(v)
		}

		vStr := fmt.Sprintf("%v", v)
		loggingHubEntry.Fields[k] = vStr
	}

	hook.channel <- loggingHubEntry

	return nil
}

// Run handles time based operations
func (hook *LoggingHubHook) Run() {
	ticker := time.NewTicker(hook.config.FlushInterval)

	for {
		select {
		case logEntry := <-hook.channel:
			numRecords, _ := hook.pendingLogs.appendLog(logEntry)

			//TODO(sakreter): Add totalSize check
			if numRecords >= hook.config.BatchSizeInLines {
				hook.Flush()
			}
		case <-ticker.C:
			hook.Flush()
		}
	}
}

// Flush flushes the logs
func (hook *LoggingHubHook) Flush() {
	entries := hook.pendingLogs.flush()
	if len(entries) == 0 {
		return
	}

	b, err := json.Marshal(LoggingHubEntriesReq{
		Senders: hook.config.Senders,
		Entries: entries,
	})

	client := &http.Client{
		Timeout: defaultHTTPClientTimeout,
	}

	resp, err := client.Post(hook.config.LoggingHubURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		log.G(context.TODO()).WithError(err).Error("Error flushing logs")
	}

	if resp.StatusCode >= 300 {
		log.G(context.TODO()).WithFields(
			logrus.Fields{
				"statusCode": resp.StatusCode,
				"error":      getHTTPErrorMsg(resp),
			},
		).Errorf("Error posting logs")
	}
}

func getHTTPErrorMsg(resp *http.Response) string {
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.G(context.TODO()).WithError(err).Error("Failed to parse response body")
		return ""
	}

	return string(b)
}

func getSize(entry *LoggingHubEntry) int {
	//TODO(sakreter): Add fields to size requirement
	return len(entry.Log) +
		len(entry.Time) +
		len(entry.Level)
}

func formatData(value interface{}) (formatted interface{}) {
	switch value := value.(type) {
	case json.Marshaler:
		return value
	case error:
		return value.Error()
	case fmt.Stringer:
		return value.String()
	default:
		return value
	}
}
