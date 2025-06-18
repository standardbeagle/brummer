// Package deadlock provides deadlock detection for development and debugging
package deadlock

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// Detector monitors lock acquisition patterns to detect potential deadlocks
type Detector struct {
	enabled   atomic.Bool
	locks     sync.Map // goroutineID -> []lockInfo
	lockOrder sync.Map // lockAddr -> []lockAddr (what locks were held when acquiring this lock)
	mu        sync.Mutex
}

type lockInfo struct {
	addr       uintptr
	file       string
	line       int
	acquiredAt time.Time
	lockType   string // "RLock", "Lock", "Channel"
}

var globalDetector = &Detector{}

// Enable turns on deadlock detection (should only be used in debug/test mode)
func Enable() {
	globalDetector.enabled.Store(true)
}

// Disable turns off deadlock detection
func Disable() {
	globalDetector.enabled.Store(false)
}

// BeforeLock should be called before acquiring a lock
func BeforeLock(lock interface{}, lockType string) {
	if !globalDetector.enabled.Load() {
		return
	}

	gid := getGoroutineID()
	addr := getLockAddr(lock)
	
	// Get caller info
	_, file, line, _ := runtime.Caller(2)
	
	info := lockInfo{
		addr:       addr,
		file:       file,
		line:       line,
		acquiredAt: time.Now(),
		lockType:   lockType,
	}

	// Check for potential deadlock
	globalDetector.checkDeadlock(gid, addr)

	// Record lock acquisition
	globalDetector.recordLock(gid, info)
}

// AfterUnlock should be called after releasing a lock
func AfterUnlock(lock interface{}) {
	if !globalDetector.enabled.Load() {
		return
	}

	gid := getGoroutineID()
	addr := getLockAddr(lock)
	
	globalDetector.removeLock(gid, addr)
}

// checkDeadlock checks for potential deadlock scenarios
func (d *Detector) checkDeadlock(gid uint64, newLockAddr uintptr) {
	// Get current locks held by this goroutine
	if locksVal, ok := d.locks.Load(gid); ok {
		currentLocks := locksVal.([]lockInfo)
		
		// Check if we're trying to acquire a lock we already hold
		for _, lock := range currentLocks {
			if lock.addr == newLockAddr {
				panic(fmt.Sprintf("DEADLOCK: Goroutine %d trying to acquire lock %x it already holds\n"+
					"First acquired at: %s:%d\n"+
					"Attempting to acquire again at current location",
					gid, newLockAddr, lock.file, lock.line))
			}
		}

		// Record lock ordering
		for _, lock := range currentLocks {
			d.recordLockOrder(lock.addr, newLockAddr)
		}
	}

	// Check for circular dependencies
	d.checkCircularDependency(newLockAddr)
}

// recordLockOrder records that lockA was held when acquiring lockB
func (d *Detector) recordLockOrder(lockA, lockB uintptr) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var deps []uintptr
	if depsVal, ok := d.lockOrder.Load(lockA); ok {
		deps = depsVal.([]uintptr)
	}

	// Check if this ordering already exists
	for _, dep := range deps {
		if dep == lockB {
			return
		}
	}

	deps = append(deps, lockB)
	d.lockOrder.Store(lockA, deps)
}

// checkCircularDependency checks if acquiring newLock would create a cycle
func (d *Detector) checkCircularDependency(newLock uintptr) {
	visited := make(map[uintptr]bool)
	path := make([]uintptr, 0)

	var checkCycle func(lock uintptr) bool
	checkCycle = func(lock uintptr) bool {
		if visited[lock] {
			// Found a cycle
			for i, l := range path {
				if l == lock {
					cycle := append(path[i:], lock)
					panic(fmt.Sprintf("DEADLOCK: Circular lock dependency detected: %v", cycle))
				}
			}
			return false
		}

		visited[lock] = true
		path = append(path, lock)
		defer func() { path = path[:len(path)-1] }()

		if depsVal, ok := d.lockOrder.Load(lock); ok {
			deps := depsVal.([]uintptr)
			for _, dep := range deps {
				if dep == newLock {
					// This would create a cycle
					cycle := append(path, newLock)
					panic(fmt.Sprintf("DEADLOCK: Lock order violation would create cycle: %v", cycle))
				}
				checkCycle(dep)
			}
		}

		return false
	}

	// Start checking from all locks that depend on newLock
	d.lockOrder.Range(func(key, value interface{}) bool {
		lockAddr := key.(uintptr)
		deps := value.([]uintptr)
		for _, dep := range deps {
			if dep == newLock {
				visited = make(map[uintptr]bool)
				path = make([]uintptr, 0)
				checkCycle(lockAddr)
			}
		}
		return true
	})
}

// recordLock records that a goroutine acquired a lock
func (d *Detector) recordLock(gid uint64, info lockInfo) {
	var locks []lockInfo
	if locksVal, ok := d.locks.Load(gid); ok {
		locks = locksVal.([]lockInfo)
	}
	locks = append(locks, info)
	d.locks.Store(gid, locks)
}

// removeLock records that a goroutine released a lock
func (d *Detector) removeLock(gid uint64, addr uintptr) {
	if locksVal, ok := d.locks.Load(gid); ok {
		locks := locksVal.([]lockInfo)
		for i, lock := range locks {
			if lock.addr == addr {
				// Remove this lock
				locks = append(locks[:i], locks[i+1:]...)
				if len(locks) == 0 {
					d.locks.Delete(gid)
				} else {
					d.locks.Store(gid, locks)
				}
				return
			}
		}
	}
}

// getGoroutineID extracts the current goroutine ID
func getGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Parse goroutine ID from stack trace
	// Format: "goroutine 123 [...]"
	var gid uint64
	fmt.Sscanf(string(buf[:n]), "goroutine %d", &gid)
	return gid
}

// getLockAddr returns the address of a lock
func getLockAddr(lock interface{}) uintptr {
	switch v := lock.(type) {
	case *sync.Mutex:
		return uintptr(unsafe.Pointer(v))
	case *sync.RWMutex:
		return uintptr(unsafe.Pointer(v))
	default:
		// For channels and other types
		return reflect.ValueOf(lock).Pointer()
	}
}

// LockTimeoutDetector monitors for locks held too long
type LockTimeoutDetector struct {
	timeout  time.Duration
	checkers sync.Map // gid -> chan struct{}
}

// NewLockTimeoutDetector creates a new timeout detector
func NewLockTimeoutDetector(timeout time.Duration) *LockTimeoutDetector {
	return &LockTimeoutDetector{
		timeout: timeout,
	}
}

// MonitorLock starts monitoring a lock acquisition
func (ltd *LockTimeoutDetector) MonitorLock(lock interface{}) func() {
	if !globalDetector.enabled.Load() {
		return func() {}
	}

	gid := getGoroutineID()
	done := make(chan struct{})
	
	_, file, line, _ := runtime.Caller(1)
	
	ltd.checkers.Store(gid, done)
	
	go func() {
		select {
		case <-time.After(ltd.timeout):
			fmt.Printf("WARNING: Lock held for more than %v by goroutine %d at %s:%d\n",
				ltd.timeout, gid, file, line)
		case <-done:
			// Lock was released in time
		}
	}()

	return func() {
		close(done)
		ltd.checkers.Delete(gid)
	}
}