package task

import (
	"fmt"
	"sync/atomic"
	"time"
)

var taskIDCounter uint64

// generateTaskID 生成唯一任务 ID
func generateTaskID() string {
	id := atomic.AddUint64(&taskIDCounter, 1)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("task_%d_%d", timestamp, id)
}
