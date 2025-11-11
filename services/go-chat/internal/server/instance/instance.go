package instance

import (
	"fmt"
	"go-chat/internal/config"
	"go-chat/internal/database"
	"go-chat/internal/logging"
	"go-chat/internal/queue"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var counter uint64 = 0

type Instance struct {
	Config     *config.Config
	Logger     *logging.Logger
	Database   *database.Database
	AMQPClient *queue.AMQP
	reqSource  string
	reqAgent   string
	trace      string
	closed     bool
}

func newTrace(appname string) string {
	id := atomic.AddUint64(&counter, 1)
	t := time.Now().UTC()
	return fmt.Sprintf("%s.%08X.%08X", strings.ToUpper(appname), t.Unix(), id)
}

func New(config *config.Config, logger *logging.Logger, database *database.Database, amqpClient *queue.AMQP) *Instance {
	inst := &Instance{
		Config:     config,
		Logger:     logger,
		Database:   database,
		AMQPClient: amqpClient,
		trace:      newTrace(config.AppName),
		closed:     false,
	}

	return inst
}

func (inst *Instance) GetLogger() *logging.Logger {
	if inst != nil {
		return inst.Logger
	}
	return nil
}

func (inst *Instance) GetSource() string {
	if inst != nil {
		return inst.reqSource
	}
	return ""
}

func (inst *Instance) SetSource(source string) {
	if inst != nil {
		inst.reqSource = source
	}
}

func (inst *Instance) SetAgent(agent string) {
	if inst != nil {
		inst.reqAgent = agent
	}
}

func (inst *Instance) Trace() string {
	if inst != nil {
		return inst.trace
	}
	return ""
}

func GetFromHttpHandler(r *http.Request) *Instance {
	if r == nil {
		return nil
	}
	inst, ok := r.Context().Value("instance").(*Instance)
	if ok == false || inst == nil {
		return nil
	}
	return inst
}
