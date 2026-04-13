package mcptools

import (
	"fmt"
	"sync"
	"testing"
)

const testAssessID = "assess-test-001"

func TestPool_RegisterAndGet(t *testing.T) {
	pool := NewConnectorPool()
	conn := safeConnector()

	pool.Register(testAssessID, conn)

	got, err := pool.Get(testAssessID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != conn {
		t.Error("returned connector is not the registered one")
	}
}

func TestPool_GetNotFound(t *testing.T) {
	pool := NewConnectorPool()

	_, err := pool.Get("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

func TestPool_Remove(t *testing.T) {
	pool := NewConnectorPool()
	pool.Register(testAssessID, safeConnector())

	pool.Remove(testAssessID)

	_, err := pool.Get(testAssessID)
	if err == nil {
		t.Fatal("expected error after Remove, got nil")
	}
}

func TestPool_RemoveNonExistent(t *testing.T) {
	pool := NewConnectorPool()
	// 删除不存在的 key 不应 panic
	pool.Remove("does-not-exist")
}

func TestPool_OverwriteRegister(t *testing.T) {
	pool := NewConnectorPool()
	conn1 := safeConnector()
	conn2 := vulnerableConnector()

	pool.Register(testAssessID, conn1)
	pool.Register(testAssessID, conn2) // 覆盖

	got, err := pool.Get(testAssessID)
	if err != nil {
		t.Fatal(err)
	}
	if got != conn2 {
		t.Error("expected second connector (overwrite), got first")
	}
}

func TestPool_ConcurrentReadWrite(t *testing.T) {
	pool := NewConnectorPool()
	const workers = 50

	var wg sync.WaitGroup
	wg.Add(workers * 3)

	// 并发写入
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			pool.Register(fmt.Sprintf("id-%d", i), safeConnector())
		}(i)
	}

	// 并发读取
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			_, _ = pool.Get(fmt.Sprintf("id-%d", i)) // 忽略 not-found 错误
		}(i)
	}

	// 并发删除
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer wg.Done()
			pool.Remove(fmt.Sprintf("id-%d", i))
		}(i)
	}

	wg.Wait()
	// 不 panic、不 deadlock 即通过（配合 -race 检查数据竞争）
}

func TestPool_MultipleAssessments(t *testing.T) {
	pool := NewConnectorPool()
	ids := []string{"a1", "b2", "c3", "d4"}

	for _, id := range ids {
		pool.Register(id, safeConnector())
	}

	for _, id := range ids {
		if _, err := pool.Get(id); err != nil {
			t.Errorf("expected connector for id %s, got error: %v", id, err)
		}
	}

	pool.Remove("b2")
	if _, err := pool.Get("b2"); err == nil {
		t.Error("expected error after remove b2, got nil")
	}
	// 其他 id 不受影响
	for _, id := range []string{"a1", "c3", "d4"} {
		if _, err := pool.Get(id); err != nil {
			t.Errorf("connector for id %s unexpectedly removed", id)
		}
	}
}
