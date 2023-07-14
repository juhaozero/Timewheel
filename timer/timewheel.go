package timer

import (
	"container/list"
	"context"
	"errors"
	"log"
	"math"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// TimeWheel 核心结构体
type TimeWheel struct {
	// 时间轮盘的精度
	interval time.Duration
	// 时间轮每个位置存储的Task列表
	slots  []*list.List
	ticker *time.Ticker
	// 时间轮当前的位置
	currentPos        int
	slotNums          int
	addTaskChannel    chan *Task
	removeTaskChannel chan *Task
	workerChan        chan Job
	workerNum         int
	cancel            context.CancelFunc
	autoId            uint64
	// Map结构来存储Task对象，key是Task.key，value是Task在双向链表中的存储对象
	taskRecords *sync.Map
	isRunning   bool
}

// Job 需要执行的Job的函数结构体
type Job func()

// Task 时间轮盘上需要执行的任务
type Task struct {
	// 用来标识task对象，是唯一的
	timerId uint64
	// 任务周期
	interval time.Duration
	// 任务在轮盘的位置
	pos int
	// 任务需要在轮盘走多少圈才能执行
	circle int
	job    Job
	// 任务需要执行的次数，如果需要一直执行，设置成-1
	times int
}

var tw *TimeWheel
var once sync.Once

// NewTimeWheel 用来实现TimeWheel的单例模式
func NewTimeWheel(interval time.Duration, slotNums, workerNum int) (*TimeWheel, error) {
	if interval <= 0 || slotNums <= 0 || workerNum <= 0 {
		return nil, errors.New("param err")
	}
	once.Do(func() {
		tw = New(interval, slotNums, workerNum)
	})
	return tw, nil
}

// New 初始化一个TimeWheel对象
func New(interval time.Duration, slotNums, workerNum int) *TimeWheel {

	tw := &TimeWheel{
		interval:          interval,
		slots:             make([]*list.List, slotNums),
		currentPos:        0,
		slotNums:          slotNums,
		addTaskChannel:    make(chan *Task),
		removeTaskChannel: make(chan *Task),
		workerChan:        make(chan Job),
		workerNum:         workerNum,
		autoId:            0,
		taskRecords:       &sync.Map{},
		isRunning:         false,
	}

	tw.initSlots()
	return tw
}

// Start 启动时间轮
func (tw *TimeWheel) Start() {
	tw.ticker = time.NewTicker(tw.interval)
	ctx, cancel := context.WithCancel(context.Background())
	tw.cancel = cancel
	go tw.start(ctx)
	for i := 0; i < tw.workerNum; i++ {
		go tw.wokerRoutine(ctx)
	}
	tw.isRunning = true
}

// Stop 关闭时间轮
func (tw *TimeWheel) Stop() {
	tw.cancel()
	tw.isRunning = false
}

// IsRunning 检查一下时间轮盘的是否在正常运行
func (tw *TimeWheel) IsRunning() bool {
	return tw.isRunning
}

// AddTask 向时间轮盘添加任务的开放函数
// #interval    任务的周期
// #autoId        任务的key，自增唯一
// # times  执行次数
func (tw *TimeWheel) AddTimer(interval time.Duration, times int, job Job) uint64 {
	if interval <= 0 || times == 0 {
		return 0
	}
	//?不可能吧
	if tw.autoId == math.MaxUint64 {
		tw.autoId = 0
	}

	task := &Task{
		timerId:  atomic.AddUint64(&tw.autoId, 1),
		interval: interval,
		job:      job,
		times:    times,
	}
	tw.addTaskChannel <- task
	return task.timerId
}

// RemoveTask 从时间轮盘删除任务的公共函数
func (tw *TimeWheel) RemoveTimer(key uint64) {
	//检查该Task是否存在
	val, ok := tw.taskRecords.Load(key)
	if !ok {
		return
	}
	if val == nil {
		return
	}
	t := val.(*list.Element)
	if t == nil {
		return
	}
	task := t.Value.(*Task)
	tw.removeTaskChannel <- task

}

// 初始化时间轮盘，每个轮盘上的卡槽用一个双向队列表示，便于插入和删除
func (tw *TimeWheel) initSlots() {
	for i := 0; i < tw.slotNums; i++ {
		tw.slots[i] = list.New()
	}
}

// 启动时间轮
func (tw *TimeWheel) start(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("timeWheel start recovered err: %s\n", r)
			log.Printf("timeWheel start recovered stack: %s\n", debug.Stack())
			// 重新启动工作routine

			go tw.start(ctx)

		}
	}()
	d := ctx.Done()
	for {
		select {
		case <-tw.ticker.C:
			tw.checkAndRunTask()
		case task := <-tw.addTaskChannel:
			tw.addTask(task)
		case task := <-tw.removeTaskChannel:
			tw.removeTask(task)
		case <-d:
			tw.ticker.Stop()
			return
		}
	}
}

// wokerRoutine 工作routine
func (tw *TimeWheel) wokerRoutine(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("timeWheel wokerRoutine recovered err: %s\n", r)
			log.Printf("timeWheel wokerRoutine recovered stack: %s\n", debug.Stack())
			// 重新启动工作routine
			go tw.wokerRoutine(ctx)

		}
	}()
	var (
		d = ctx.Done()
	)
	for {
		select {
		case <-d:
			return
		case execFunc := <-tw.workerChan:
			execFunc()
		}
	}
}

// 检查该轮盘点位上的Task，看哪个需要执行
func (tw *TimeWheel) checkAndRunTask() {

	// 获取该轮盘位置的双向链表
	currentList := tw.slots[tw.currentPos]

	if currentList != nil {
		for item := currentList.Front(); item != nil; {
			task := item.Value.(*Task)
			// 如果圈数>0，表示还没到执行时间，更新圈数
			if task.circle > 0 {
				task.circle--
				item = item.Next()
				continue
			}

			if task.job != nil && task.times != 0 {
				//压入工作池
				tw.workerChan <- task.job
			}
			// 执行完成以后，将该任务从时间轮盘删除
			next := item.Next()

			tw.taskRecords.Delete(task.timerId)

			currentList.Remove(item)
			item = next

			// 重新添加任务到时间轮盘，用Task.interval来获取下一次执行的轮盘位置
			// 如果times==0,说明已经完成执行周期,不需要再添加任务回时间轮
			if task.times == -1 {
				tw.addTask(task)
			} else {
				task.times--
				if task.times > 0 {
					tw.addTask(task)
				}
			}
		}
	}

	// 轮盘前进一步
	if tw.currentPos == tw.slotNums-1 {
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}
}

// addTask 添加任务
func (tw *TimeWheel) addTask(task *Task) {
	var pos, circle int
	pos, circle = tw.getPosAndCircleByInterval(task.interval)

	task.circle = circle
	task.pos = pos

	element := tw.slots[pos].PushBack(task)
	tw.taskRecords.Store(task.timerId, element)
}

// removeTask 删除任务
func (tw *TimeWheel) removeTask(task *Task) {
	// 从map结构中删除
	val, _ := tw.taskRecords.Load(task.timerId)
	if val == nil {
		return
	}
	tw.taskRecords.Delete(task.timerId)
	// 通过TimeWheel.slots获取任务的
	currentList := tw.slots[task.pos]
	data := val.(*list.Element)
	if data != nil {
		currentList.Remove(data)
	}
}

// getPosAndCircleByInterval 周期获取
func (tw *TimeWheel) getPosAndCircleByInterval(delay time.Duration) (int, int) {
	d := int(delay)
	interval := int(tw.interval)
	circle := d / interval / tw.slotNums
	pos := (tw.currentPos + d/interval) % tw.slotNums

	return pos, circle
}
