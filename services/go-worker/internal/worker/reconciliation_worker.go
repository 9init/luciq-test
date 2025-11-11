package worker

import (
	"context"
	"fmt"
	"go-worker/internal/database"
	"go-worker/internal/logging"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type ReconciliationWorker struct {
	repo         *database.Repository
	redis        *redis.Client
	logger       *logging.Logger
	ticker       *time.Ticker
	stopChan     chan struct{}
	syncInterval time.Duration
	lockKey      string
	lockTTL      time.Duration
}

type reconcileConfig struct {
	pattern       string
	entityName    string
	updateFunc    func(uint, int) error
	extractIDFunc func(string) (uint, error)
}

func NewReconciliationWorker(db *database.Database, logger *logging.Logger) *ReconciliationWorker {
	w := &ReconciliationWorker{
		repo:         database.NewRepository(db.MySqlDB),
		redis:        db.RedisDB,
		logger:       logger.WithPrefix("ReconciliationWorker"),
		syncInterval: 15 * time.Second,
		ticker:       time.NewTicker(15 * time.Second),
		stopChan:     make(chan struct{}),
		lockKey:      "lock:reconciliation",
		lockTTL:      15 * time.Second,
	}

	go w.start()

	return w
}

func (w *ReconciliationWorker) start() {
	w.logger.Info("Started with interval: %v", w.syncInterval)

	for {
		select {
		case <-w.ticker.C:
			if err := w.reconcile(); err != nil {
				w.logger.Error("Reconciliation failed: %v", err)
			}
		case <-w.stopChan:
			w.logger.Info("Stopping...")
			return
		}
	}
}

func (w *ReconciliationWorker) reconcile() error {
	ctx := context.Background()
	acquired, err := w.acquireLock(ctx)
	if err != nil {
		w.logger.Error("Failed to acquire lock: %v", err)
		return err
	}

	if !acquired {
		w.logger.Info("Another instance is reconciling, skipping")
		return nil
	}
	defer w.releaseLock(ctx)
	// w.logger.Info("Acquired reconciliation lock, starting reconciliation")

	appChatConfig := reconcileConfig{
		pattern:    "delta:app:*:chats",
		entityName: "application",
		updateFunc: w.repo.IncrementApplicationChatCount,
		extractIDFunc: func(key string) (uint, error) {
			// Extract from: delta:app:123:chats
			parts := strings.Split(key, ":")
			if len(parts) != 4 {
				return 0, fmt.Errorf("invalid key format: %s", key)
			}
			id, err := strconv.ParseUint(parts[2], 10, 64)
			return uint(id), err
		},
	}

	chatMessageConfig := reconcileConfig{
		pattern:    "delta:chat:*:messages",
		entityName: "chat",
		updateFunc: w.repo.IncrementChatMessageCount,
		extractIDFunc: func(key string) (uint, error) {
			// Extract from: delta:chat:456:messages
			parts := strings.Split(key, ":")
			if len(parts) != 4 {
				return 0, fmt.Errorf("invalid key format: %s", key)
			}
			id, err := strconv.ParseUint(parts[2], 10, 64)
			return uint(id), err
		},
	}

	if err := w.reconcileEntity(ctx, appChatConfig); err != nil {
		w.logger.Error("Failed to reconcile app chat counts: %v", err)
	}
	if err := w.reconcileEntity(ctx, chatMessageConfig); err != nil {
		w.logger.Error("Failed to reconcile chat message counts: %v", err)
	}

	return nil
}

func (w *ReconciliationWorker) reconcileEntity(ctx context.Context, config reconcileConfig) error {
	keys, err := w.scanKeys(ctx, config.pattern)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}

	w.logger.Info("Reconciling %d %s counts", len(keys), config.entityName)
	for _, key := range keys {
		if err := w.processKey(ctx, key, config); err != nil {
			w.logger.Error("Failed to reconcile %s: %v", key, err)
		}
	}

	return nil
}

func (w *ReconciliationWorker) processKey(ctx context.Context, key string, config reconcileConfig) error {
	delta, err := w.atomicGetDel(ctx, key)
	if err != nil {
		return err
	}
	if delta == 0 {
		return nil
	}

	entityID, err := config.extractIDFunc(key)
	if err != nil {
		return err
	}

	if err := config.updateFunc(entityID, delta); err != nil {
		w.redis.IncrBy(ctx, key, int64(delta))
		return fmt.Errorf("failed to update %s %d: %v", config.entityName, entityID, err)
	}
	w.logger.Info("Updated %s %d count by %d", config.entityName, entityID, delta)

	return nil
}

func (w *ReconciliationWorker) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var cursor uint64
	var keys []string

	for {
		batch, newCursor, err := w.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

func (w *ReconciliationWorker) atomicGetDel(ctx context.Context, key string) (int, error) {
	// I can simply use GETDEL, but its atomicity is not guaranteed across multiple instances
	// And I like defensive programming :)
	script := `
        local value = redis.call('GET', KEYS[1])
        if value then
            redis.call('DEL', KEYS[1])
            return value
        end
        return nil
    `

	result, err := w.redis.Eval(ctx, script, []string{key}).Result()
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}

	delta, err := strconv.Atoi(fmt.Sprintf("%v", result))
	if err != nil {
		return 0, err
	}

	return delta, nil
}

func (w *ReconciliationWorker) acquireLock(ctx context.Context) (bool, error) {
	result, err := w.redis.SetNX(ctx, w.lockKey, "1", w.lockTTL).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

func (w *ReconciliationWorker) releaseLock(ctx context.Context) {
	if err := w.redis.Del(ctx, w.lockKey).Err(); err != nil {
		w.logger.Error("Failed to release lock: %v", err)
	}
}

func (w *ReconciliationWorker) Stop() {
	w.logger.Info("Stopping reconciliation worker")
	close(w.stopChan)
	w.ticker.Stop()

	ctx := context.Background()
	acquired, _ := w.acquireLock(ctx)
	if acquired {
		defer w.releaseLock(ctx)
		if err := w.reconcile(); err != nil {
			w.logger.Error("Final reconciliation failed: %v", err)
		}
	}
}
